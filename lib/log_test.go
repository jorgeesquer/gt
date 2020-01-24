package lib

// import (
// 	"fmt"
// 	"github.com/gtlang/gt/lib/x/logdb"
// 	"testing"
// 	"time"
// )

// func TestLog(t *testing.T) {
// 	fs := NewVirtualFS()

// 	db := &logDB{
// 		db: logdb.New("logs", fs),
// 	}

// 	db.db.Save("access", "GET %d", 200)

// 	fs.PrintPaths()

// 	s := db.db.Query("access", time.Now().Add(-10*time.Second), time.Now(), 0, 0)

// 	for s.Scan() {
// 		fmt.Println(s.Data())
// 	}
// }
