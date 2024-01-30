package db

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// Test_WALDatabaseCheckpointOKNoWAL tests that a checkpoint succeeds
// even when no WAL file exists.
func Test_WALDatabaseCheckpointOKNoWAL(t *testing.T) {
	path := mustTempFile()
	defer os.Remove(path)

	db, err := Open(path, false, true)
	if err != nil {
		t.Fatalf("failed to open database in WAL mode: %s", err.Error())
	}
	if !db.WALEnabled() {
		t.Fatalf("WAL mode not enabled")
	}
	if fileExists(db.WALPath()) {
		t.Fatalf("WAL file exists when no writes have happened")
	}
	defer db.Close()
	if err := db.Checkpoint(CheckpointTruncate); err != nil {
		t.Fatalf("failed to checkpoint database in WAL mode with non-existent WAL: %s", err.Error())
	}
}

// Test_WALDatabaseCheckpointOKDelete tests that a checkpoint returns no error
// even when the database is opened in DELETE mode.
func Test_WALDatabaseCheckpointOKDelete(t *testing.T) {
	path := mustTempFile()
	defer os.Remove(path)

	db, err := Open(path, false, false)
	if err != nil {
		t.Fatalf("failed to open database in DELETE mode: %s", err.Error())
	}
	if db.WALEnabled() {
		t.Fatalf("WAL mode enabled")
	}
	defer db.Close()
	if err := db.Checkpoint(CheckpointTruncate); err != nil {
		t.Fatalf("failed to checkpoint database in DELETE mode: %s", err.Error())
	}
}

// Test_WALDatabaseCheckpoint_Restart tests that a checkpoint restart
// returns no error and that the WAL file is not modified even though
// all the WAL pages are copied to the database file. Then Truncate
// is called and the WAL file is deleted.
func Test_WALDatabaseCheckpoint_RestartTruncate(t *testing.T) {
	path := mustTempFile()
	defer os.Remove(path)
	db, err := Open(path, false, true)
	if err != nil {
		t.Fatalf("failed to open database in WAL mode: %s", err.Error())
	}
	defer db.Close()

	_, err = db.ExecuteStringStmt(`CREATE TABLE foo (id INTEGER NOT NULL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("failed to execute on single node: %s", err.Error())
	}
	for i := 0; i < 50; i++ {
		_, err := db.ExecuteStringStmt(`INSERT INTO foo(name) VALUES("fiona")`)
		if err != nil {
			t.Fatalf("failed to execute INSERT on single node: %s", err.Error())
		}
	}

	walPreBytes := mustReadBytes(db.WALPath())
	if err := db.Checkpoint(CheckpointRestart); err != nil {
		t.Fatalf("failed to checkpoint database: %s", err.Error())
	}
	walPostBytes := mustReadBytes(db.WALPath())
	if !bytes.Equal(walPreBytes, walPostBytes) {
		t.Fatalf("wal file should be unchanged after checkpoint restart")
	}

	// query the data to make sure all is well.
	rows, err := db.QueryStringStmt(`SELECT COUNT(*) FROM foo`)
	if err != nil {
		t.Fatalf("failed to execute query on single node: %s", err.Error())
	}
	if exp, got := `[{"columns":["COUNT(*)"],"types":["integer"],"values":[[50]]}]`, asJSON(rows); exp != got {
		t.Fatalf("expected %s, got %s", exp, got)
	}

	if err := db.Checkpoint(CheckpointTruncate); err != nil {
		t.Fatalf("failed to checkpoint database: %s", err.Error())
	}
	sz, err := fileSize(db.WALPath())
	if err != nil {
		t.Fatalf("wal file should be deleted after checkpoint truncate")
	}
	if sz != 0 {
		t.Fatalf("wal file should be zero length after checkpoint truncate")
	}

	// query the data to make sure all is still well.
	rows, err = db.QueryStringStmt(`SELECT COUNT(*) FROM foo`)
	if err != nil {
		t.Fatalf("failed to execute query on single node: %s", err.Error())
	}
	if exp, got := `[{"columns":["COUNT(*)"],"types":["integer"],"values":[[50]]}]`, asJSON(rows); exp != got {
		t.Fatalf("expected %s, got %s", exp, got)
	}
}

// Test_WALDatabaseCheckpoint_RestartTimeout tests that a restart checkpoint
// does time out as expected if there is a long running read.
func Test_WALDatabaseCheckpoint_RestartTimeout(t *testing.T) {
	path := mustTempFile()
	defer os.Remove(path)
	db, err := Open(path, false, true)
	if err != nil {
		t.Fatalf("failed to open database in WAL mode: %s", err.Error())
	}
	defer db.Close()

	_, err = db.ExecuteStringStmt(`CREATE TABLE foo (id INTEGER NOT NULL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("failed to execute on single node: %s", err.Error())
	}
	for i := 0; i < 50; i++ {
		_, err := db.ExecuteStringStmt(`INSERT INTO foo(name) VALUES("fiona")`)
		if err != nil {
			t.Fatalf("failed to execute INSERT on single node: %s", err.Error())
		}
	}

	blockingDB, err := Open(path, false, true)
	if err != nil {
		t.Fatalf("failed to open blocking database in WAL mode: %s", err.Error())
	}
	defer blockingDB.Close()
	_, err = blockingDB.QueryStringStmt(`BEGIN TRANSACTION`)
	if err != nil {
		t.Fatalf("failed to execute query on single node: %s", err.Error())
	}
	rows, err := blockingDB.QueryStringStmt(`SELECT COUNT(*) FROM foo`)
	if err != nil {
		t.Fatalf("failed to execute query on single node: %s", err.Error())
	}
	if exp, got := `[{"columns":["COUNT(*)"],"types":["integer"],"values":[[50]]}]`, asJSON(rows); exp != got {
		t.Fatalf("expected %s, got %s", exp, got)
	}

	if err := db.CheckpointWithTimeout(CheckpointRestart, 250*time.Millisecond); err == nil {
		t.Fatal("expected error due to failure to checkpoint")
	}

	blockingDB.Close()
	if err := db.CheckpointWithTimeout(CheckpointRestart, 250*time.Millisecond); err != nil {
		t.Fatalf("failed to checkpoint database: %s", err.Error())
	}
}

// Test_WALDatabaseCheckpoint_TruncateTimeout tests that a truncate checkpoint
// does time out as expected if there is a long running read. It also confirms
// that the WAL file is not modified as a result of this failure.
func Test_WALDatabaseCheckpoint_TruncateTimeout(t *testing.T) {
	path := mustTempFile()
	defer os.Remove(path)
	db, err := Open(path, false, true)
	if err != nil {
		t.Fatalf("failed to open database in WAL mode: %s", err.Error())
	}
	defer db.Close()

	_, err = db.ExecuteStringStmt(`CREATE TABLE foo (id INTEGER NOT NULL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("failed to execute on single node: %s", err.Error())
	}
	for i := 0; i < 50; i++ {
		_, err := db.ExecuteStringStmt(`INSERT INTO foo(name) VALUES("fiona")`)
		if err != nil {
			t.Fatalf("failed to execute INSERT on single node: %s", err.Error())
		}
	}

	preWALBytes := mustReadBytes(db.WALPath())
	blockingDB, err := Open(path, false, true)
	if err != nil {
		t.Fatalf("failed to open blocking database in WAL mode: %s", err.Error())
	}
	defer blockingDB.Close()
	_, err = blockingDB.QueryStringStmt(`BEGIN TRANSACTION`)
	if err != nil {
		t.Fatalf("failed to execute query on single node: %s", err.Error())
	}
	rows, err := blockingDB.QueryStringStmt(`SELECT COUNT(*) FROM foo`)
	if err != nil {
		t.Fatalf("failed to execute query on single node: %s", err.Error())
	}
	if exp, got := `[{"columns":["COUNT(*)"],"types":["integer"],"values":[[50]]}]`, asJSON(rows); exp != got {
		t.Fatalf("expected %s, got %s", exp, got)
	}

	if err := db.CheckpointWithTimeout(CheckpointTruncate, 250*time.Millisecond); err == nil {
		t.Fatal("expected error due to failure to checkpoint")
	}
	postWALBytes := mustReadBytes(db.WALPath())
	if !bytes.Equal(preWALBytes, postWALBytes) {
		t.Fatalf("wal file should be unchanged after checkpoint failure")
	}

	blockingDB.Close()
	if err := db.CheckpointWithTimeout(CheckpointTruncate, 250*time.Millisecond); err != nil {
		t.Fatalf("failed to checkpoint database: %s", err.Error())
	}
	if mustFileSize(db.WALPath()) != 0 {
		t.Fatalf("wal file should be zero length after checkpoint truncate")
	}
}

func mustReadBytes(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}