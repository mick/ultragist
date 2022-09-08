package sqlite3store

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/psanford/sqlite3vfs"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"
)

func getObjectSize(bucket string, key string) (int64, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return 0, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*60)
	defer cancel()

	query := &storage.Query{Prefix: key + "/"}

	var size int64
	it := client.Bucket(bucket).Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		size += attrs.Size
	}

	return size, nil
}

type GcsVFS struct {
	CacheHandler CacheHandler
	RoundTripper http.RoundTripper
}

type CacheHandler interface {
	Get(key interface{}) (value interface{}, ok bool)
	Add(key interface{}, value interface{})
}

func (vfs *GcsVFS) Open(name string, flags sqlite3vfs.OpenFlag) (sqlite3vfs.File, sqlite3vfs.OpenFlag, error) {
	fmt.Println("Open", name, flags)
	u, err := url.Parse(name)
	if err != nil {
		panic(err)
	}
	tf := &gcsFile{
		bucket:        u.Host,
		key:           u.Path[1:],
		name:          name,
		cacheHandler:  vfs.CacheHandler,
		roundTripper:  vfs.RoundTripper,
		chunkSize:     4096 * 1, //this need to fit the page boundaries, default 4096 max 65536,
		pendingWrites: sync.Map{},
	}

	return tf, flags, nil
}

func (vfs *GcsVFS) Delete(name string, dirSync bool) error {
	fmt.Println("Delete", name, dirSync)
	return sqlite3vfs.ReadOnlyError
}

func (vfs *GcsVFS) Access(name string, flag sqlite3vfs.AccessFlag) (bool, error) {
	fmt.Println("access", name, flag)
	if strings.HasSuffix(name, "-wal") || strings.HasSuffix(name, "-journal") {
		return false, nil
	}

	return true, nil
}

func (vfs *GcsVFS) FullPathname(name string) string {
	return name
}

type gcsFile struct {
	bucket        string
	key           string
	name          string
	cacheHandler  CacheHandler
	roundTripper  http.RoundTripper
	chunkSize     int64
	pendingWrites sync.Map
}

func (tf *gcsFile) Close() error {
	return nil
}

// func (tf *gcsFile) client() *http.Client {
// 	if tf.roundTripper == nil {
// 		return http.DefaultClient
// 	}
// 	return &http.Client{
// 		Transport: tf.roundTripper,
// 	}
// }

// var hits = 0
// var misses = 0

func (tf *gcsFile) ReadAt(p []byte, off int64) (int, error) {
	fmt.Println("read at", off, len(p))
	offStart := off % tf.chunkSize
	chunkStart := tf.chunkSize * int64(math.Floor(float64(off)/float64(tf.chunkSize)))

	// 	if tf.cacheHandler != nil {
	// 		buf, ok := tf.cacheHandler.Get(fmt.Sprintf("%s-%d", tf.name, chunkStart))
	// 		if ok {
	// 			hits += 1
	// 			// fmt.Printf("Cache hit: %v\n", fmt.Sprintf("%s-%d", tf.name, chunkStart))
	//
	// 			copy(p, buf.([]byte)[offStart:])
	// 			// fmt.Printf("P cache Bytes %v to %v of blob \n", chunkStart, chunkStart+tf.chunkSize)
	// 			return len(p), nil
	// 		} else {
	// 			misses += 1
	// 		}
	// 		// fmt.Printf("Cache miss: %v hits: %v\n", misses, hits)
	// 	}
	// fmt.Printf("ReadAt: %v - %v\n", off, len(p))

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return 0, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	partNumber := getPartNumber(tf.chunkSize, off)
	partKey := getPartKey(tf.key, partNumber)
	//todo what about end of the file? will that just work out? probably...
	fmt.Printf("read key %v %v\n", tf.bucket, partKey)
	fmt.Printf("ReadAt chrunk start %v offset %v len %v\n", chunkStart, offStart, len(p))
	rc, err := client.Bucket(tf.bucket).Object(partKey).NewReader(ctx)
	if err != nil && err.Error() == "storage: object doesn't exist" {

		return 0, nil
		// return 0, fmt.Errorf("Object(%q).NewReader: %v", tf.key, err)
	} else if err != nil {
		fmt.Printf("Object(%q).NewReader: %v", tf.key, err)
		return 0, fmt.Errorf("Object(%q).NewReader: %v", tf.key, err)
	}
	defer rc.Close()
	fullbuf := make([]byte, tf.chunkSize)
	n, err := io.ReadFull(rc, fullbuf)
	if err != nil {
		fmt.Printf("io.ReadFull: %v", err)
		return 0, err
	}

	copy(p, fullbuf[offStart:])
	if tf.cacheHandler != nil {
		tf.cacheHandler.Add(fmt.Sprintf("%s-%d", tf.name, chunkStart), fullbuf)
	}
	// fmt.Printf("P Bytes %v to %v of blob \n", off, off+int64(n))
	return n, nil
}

func getPartNumber(chunkSize int64, offset int64) int64 {
	chunkStart := chunkSize * int64(math.Floor(float64(offset)/float64(chunkSize)))
	partNumber := chunkStart / chunkSize
	return partNumber
}
func getPartKey(key string, partNumber int64) string {
	return fmt.Sprintf("%s/part-%09d", key, partNumber)
}

func (tf *gcsFile) WriteAt(b []byte, off int64) (n int, err error) {
	fmt.Println("WriteAt off, len", off, len(b))

	// offStart := off % tf.chunkSize
	partNumber := getPartNumber(tf.chunkSize, off)
	partData := make([]byte, len(b))
	copy(partData, b)
	tf.pendingWrites.Store(partNumber, partData)

	fmt.Println("bytes written", len(partData))
	return 0, nil
}

func (tf *gcsFile) Truncate(size int64) error {
	fmt.Println("truncate", size)
	return sqlite3vfs.ReadOnlyError
}

func (tf *gcsFile) Sync(flag sqlite3vfs.SyncType) error {
	fmt.Println("Sync", flag)
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()
	// var wg sync.WaitGroup
	// wg.Add(len(tf.pendingWrites))

	var eg errgroup.Group
	tf.pendingWrites.Range(func(partNumber, data interface{}) bool {
		pn := partNumber.(int64)
		d := data.([]byte)
		partKey := getPartKey(tf.key, pn)
		fmt.Printf("write key %v %v\n", tf.bucket, partKey)
		eg.Go(func() error {
			// defer wg.Done()
			fmt.Printf("Writing partnumber %d data %v to %v\n", pn, d[:10], partKey)
			wc := client.Bucket(tf.bucket).Object(partKey).NewWriter(ctx)
			if _, err := wc.Write(d); err != nil {
				return err
			}
			if err := wc.Close(); err != nil {
				return err
			}
			tf.pendingWrites.Delete(pn)
			return nil
		})
		return true
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	fmt.Println("Sync done")
	return nil
}

// var invalidContentRangeErr = errors.New("invalid Content-Range response")

func (tf *gcsFile) FileSize() (int64, error) {
	fmt.Println("file size")
	// this will need to change to either list all the pages and add them up, or
	// fetch that metadata from somewhere
	size, err := getObjectSize(tf.bucket, tf.key)
	if err != nil && err.Error() == "storage: object doesn't exist" {
		fmt.Printf("object doesnt exist yet")
		return 0, nil
	} else if err != nil {
		fmt.Printf("getObjectSize: %v", err)
		return 0, err
	}

	return size, nil
}

func (tf *gcsFile) Lock(elock sqlite3vfs.LockType) error {
	fmt.Println("Lock requested", elock)
	return nil
}

func (tf *gcsFile) Unlock(elock sqlite3vfs.LockType) error {
	fmt.Println("unlock requested", elock)
	return nil
}

func (tf *gcsFile) CheckReservedLock() (bool, error) {
	fmt.Println("check reserved lock")
	return false, nil
}

func (tf *gcsFile) SectorSize() int64 {
	fmt.Println("sector size")
	return 0
}

func (tf *gcsFile) DeviceCharacteristics() sqlite3vfs.DeviceCharacteristic {
	return sqlite3vfs.IocapAtomic64K
}
