package ultragist

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var sharedDBConnection *sql.DB

func getDB(dbpath string, readonly bool) *sql.DB {
	if sharedDBConnection != nil {
		return sharedDBConnection
	}
	params := "mode=rwc"
	connstr := fmt.Sprintf("%s?%s", dbpath, params)
	var err error
	sharedDBConnection, err = sql.Open("sqlite3", connstr)
	if err != nil {
		log.Fatalf("open db err: %s", err)
	}
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
