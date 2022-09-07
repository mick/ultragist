package ultragist

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

func GetKeyByFingerprint(fingerprint string) (Key, error) {
	var key Key = Key{
		Fingerprint: fingerprint,
	}
	db := getDB(true)
	stmt, err := db.Prepare("SELECT publickey, userid FROM sshkeys WHERE fingerprint = ?")
	if err != nil {
		return key, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(fingerprint).Scan(&key.PublicKey, &key.UserId)
	if err != nil {
		return key, err
	}
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.PublicKey))
	if err != nil {
		return key, err
	}
	key.pk = pk
	return key, nil

}

func WriteKey(pubKeyBytes []byte, userId string) error {

	publickey := string(pubKeyBytes)

	// Parse the key, other info ignored
	pk, _, _, _, err := ssh.ParseAuthorizedKey(pubKeyBytes)
	if err != nil {
		return err
	}

	// Get the fingerprint
	f := ssh.FingerprintSHA256(pk)

	db := getDB(false)
	// maybe this should just fail?
	stmt, err := db.Prepare(`INSERT INTO sshkeys(fingerprint, publickey, userid) VALUES(?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(f, publickey, userId)
	if err != nil {
		return err
	}
	fmt.Println("rows affected", res)
	return nil
}

type Key struct {
	Fingerprint string        `json:"fingerprint"`
	PublicKey   string        `json:"publicKey"`
	pk          ssh.PublicKey `json:"-"`
	UserId      string        `json:"userId"` // this is an id, we still have to look up the user
}
