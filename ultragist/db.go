package ultragist

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	_ "github.com/mattn/go-sqlite3"
	sqlite3store "github.com/mick/ultragist/sqlite3store"
	"github.com/psanford/sqlite3vfs"
	"google.golang.org/api/iterator"
)

var sharedDBConnection *sql.DB

func getDB(dbpath string, readonly bool) *sql.DB {
	if sharedDBConnection != nil {
		return sharedDBConnection
	}
	// dbfilepath := os.Getenv("DBPATH")
	// if dbfilepath == "" {
	// 	dbfilepath = "ultragist.db"
	// }
	// mode := "ro"
	// if !readonly {
	// 	mode = "rwc"
	// }
	// dbpath := fmt.Sprintf("file:%s?mode=%s", dbfilepath, mode)
	// db, err := sql.Open("sqlite3", dbpath)
	// if err != nil {
	// 	log.Fatalf("open db err: %s", err)
	// }
	// return db

	// if dbCache[tileset] != nil {
	// 	return dbCache[tileset]
	// }

	// bucket := "simplemap-scratch"
	// prefix := "sqlite"
	// dbname := "test"
	var params string
	if len(dbpath) > 2 && dbpath[:2] == "gs" {
		params = "vfs=gcsvfs&mode=rwc&_journal=OFF&_sync=3"
	} else {
		params = "mode=rwc"
	}

	// todo check if this file exists, so we short circuit / avoid query errors
	connstr := fmt.Sprintf("%s?%s", dbpath, params)
	var err error
	sharedDBConnection, err = sql.Open("sqlite3", connstr)
	if err != nil {
		log.Fatalf("open db err: %s", err)
	}
	// sharedDBConnection.Exec("PRAGMA page_size = 65536")

	// dbCache[tileset] = db
	return sharedDBConnection

}

func InitDB(dbpath string) error {
	db := getDB(dbpath, false)

	sqlStmt := `
	create table if not exists sshkeys (fingerprint text not null primary key, publickey text, userid text);
	create table if not exists users (userid text not null primary key, username text, email text);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return err
	}
	return nil
}

func InitSqliteVFS() error {
	vfs := sqlite3store.GcsVFS{
		// CacheHandler: cache,
	}

	err := sqlite3vfs.RegisterVFS("gcsvfs", &vfs)
	if err != nil {
		fmt.Printf("register vfs err: %s", err)
		return err
	}
	return nil
}

func DBTest(dbpath string) error {
	fmt.Println("DBTest")

	db := getDB(dbpath, false)

	sqlStmt := ""
	for i := 3000; i < 4000; i++ {

		sqlStmt += fmt.Sprintf(`INSERT INTO users (userid, username, email) VALUES ('%d', 'mick', 'email@mail.com');`, i)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(sqlStmt)
	if err != nil {

		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Println("rows affected", rows)
	err = tx.Commit()
	if err != nil {
		return err
	}

	stmt, err := db.Prepare("SELECT userid, username, email FROM users WHERE userid = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	var userid, username, email string

	err = stmt.QueryRow("3050").Scan(&userid, &username, &email)
	if err != nil {
		return err
	}
	fmt.Println(userid, username, email)

	return nil
}
func parseGCSPath(path string) (bucket, prefix string) {
	parts := strings.Split(path, "/")
	bucket = parts[2]
	prefix = strings.Join(parts[3:], "/")
	return
}
func DBExport(gcspath string, dest string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*60)
	defer cancel()

	// parse url
	bucket, prefix := parseGCSPath(gcspath)
	query := &storage.Query{Prefix: prefix + "/"}

	var names []string
	it := client.Bucket(bucket).Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		names = append(names, attrs.Name)
	}

	buck := client.Bucket(bucket)

	file, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, name := range names {
		fmt.Println(name)
		rc, err := buck.Object(name).NewReader(ctx)
		if err != nil {
			return fmt.Errorf("Object(%q).NewReader: %v", name, err)
		}
		io.Copy(file, rc)
		rc.Close()
	}

	return nil
}
