package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	log.Printf("[*] connecting to a database")

	db1, err := sql.Open("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatalf("failed to connect to a database: %v", err)
	}

	db2, err := sql.Open("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatalf("failed to connect to a database: %v", err)
	}

	log.Printf("[*] setting up a table")
	_, err = db1.Exec(`
		CREATE TABLE IF NOT EXISTS public.account_versions (
			account VARCHAR(100) PRIMARY KEY,
		    version BIGINT
	    );
		
		DELETE FROM public.account_versions;

		INSERT INTO public.account_versions (account, version)
			VALUES ('alice', 3), ('bob', 5), ('candy', 4);
	`)
	if err != nil {
		log.Fatalf("failed to set up a table: %v", err)
	}

	var wg = new(sync.WaitGroup)

	/*
		Isolation levels
		----------------
		LevelDefault
		LevelReadUncommitted
		LevelReadCommitted
		LevelWriteCommitted
		LevelRepeatableRead
		LevelSnapshot
		LevelSerializable
		LevelLinearizable
	*/

	// --- transaction 1 ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Printf("[*] tx1 is done")

		log.Printf("[*] begin tx1")
		tx1, err := db1.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		if err != nil {
			log.Fatalf("failed to begin the first transaction: %v", err)
		}

		log.Printf("[*] tx1.select-for-update")
		result, err := tx1.Query("SELECT version FROM public.account_versions WHERE account IN ('alice', 'bob') FOR UPDATE")
		if err != nil {
			log.Fatalf("failed to SELECT FOR UPDATE in the first transaction: %v", err)
		}

		defer result.Close()

		for result.Next() {
			result.Columns()
		}

		log.Printf("[*] tx1.wait-for-1s")
		time.Sleep(1 * time.Second)

		log.Printf("[*] tx1.update")
		_, err = tx1.Exec("UPDATE public.account_versions SET version=6 WHERE account IN ('alice', 'bob')")
		if err != nil {
			log.Printf("failed to execute UPDATE in the first transaction: %v", err)
			tx1.Rollback()
			return
		}

		log.Printf("[*] tx1.commit")
		err = tx1.Commit()
		if err != nil {
			log.Printf("failed to COMMIT the first transaction: %v", err)
			tx1.Rollback()
			return
		}
	}()

	// --- transaction 2 ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Printf("[*] tx2 is done")

		log.Printf("[*] begin tx2")
		tx2, err := db2.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		if err != nil {
			log.Fatalf("failed to begin the second transaction: %v", err)
		}

		log.Printf("[*] tx2.select-for-update")
		result, err := tx2.Query("SELECT version FROM public.account_versions WHERE account IN ('alice', 'bob') FOR UPDATE")
		if err != nil {
			log.Printf("failed to execute UPDATE in the second transaction: %v", err)
			tx2.Rollback()
			return
		}

		defer result.Close()

		for result.Next() {
			result.Columns()
		}

		log.Printf("[*] tx2.wait-for-2s")
		time.Sleep(2 * time.Second)

		log.Printf("[*] tx2.update")
		_, err = tx2.Exec("UPDATE public.account_versions SET version=6 WHERE account IN ('bob', 'candy')")
		if err != nil {
			log.Printf("failed to execute UPDATE in the second transaction: %v", err)
			tx2.Rollback()
			return
		}

		log.Printf("[*] tx2.commit")
		err = tx2.Commit()
		if err != nil {
			log.Printf("failed to COMMIT the second transaction: %v", err)
			tx2.Rollback()
			return
		}
	}()

	wg.Wait()

	// --- list account versions ---
	log.Printf("[*] listing account versions:")
	result, err := db2.Query("SELECT account, version FROM public.account_versions ORDER BY account ASC")
	if err != nil {
		log.Fatalf("failed to query account versions: %v", err)
	}

	defer result.Close()

	for result.Next() {
		var (
			account string
			version int64
		)

		err = result.Scan(&account, &version)
		if err != nil {
			log.Fatalf("failed to scan a row: %v", err)
		}

		log.Printf(" - %6v : %v", account, version)
	}
}
