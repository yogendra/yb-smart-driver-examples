package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/yugabyte/pgx/v4/pgxpool"
)

var numGoRoutines int = 3
var pool *pgxpool.Pool
var wg sync.WaitGroup

func startPoolExample() {
	// Create a table and insert a row
	url := fmt.Sprintf("%s&load_balance=true", baseUrl)
	pause()
	initPool(url)
	defer pool.Close()
	createTableUsingPool(url)
	printAZInfo()
	pause()

	// make connections using the url via different go routines and check load balance
	executeQueriesOnPool()
	pause()

	// add a server with a different placement zone
	fmt.Println("Adding a new server in zone rack2 ...")
	cmd := exec.Command(ybInstallPath+"/bin/yb-ctl", "add_node", "--placement_info", "cloud1.datacenter1.rack2")
	var errout bytes.Buffer
	cmd.Stderr = &errout
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Could not add a YBDB server: %s", errout)
	}
	time.Sleep(5 * time.Second)
	numGoRoutines = 12
	executeQueriesOnPool()
	printAZInfo()
	pause()

	// bring down a server and create new connections via go routines. check load balance
	fmt.Println("Stopping server 2 ...")
	cmd = exec.Command(ybInstallPath+"/bin/yb-ctl", "stop_node", "2")
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not stop a YBDB server: %s", errout)
	}
	executeQueriesOnPool()
	pause()

	// create a new pool to a new placement zone and check load balance
	// pool.Close()
	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter1.rack2", baseUrl)
	initPool(url)
	executeQueriesOnPool()
	verifyLoad(map[string]int{"127.0.0.1": 4, "127.0.0.2": 0, "127.0.0.3": 4, "127.0.0.4": numGoRoutines + 4})
	pause()

	fmt.Println("Closing the application ...")
}

func initPool(url string) {
	var err error
	fmt.Printf("Initializing pool with url %s\n", url)
	pool, err = pgxpool.Connect(context.Background(), url)
	if err != nil {
		log.Fatalf("Error initializing the pool: %s", err.Error())
	}
}

func createTableUsingPool(url string) {
	fmt.Println("Creating table using pool.Acquire() ...")
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Release()

	var dropStmt = `DROP TABLE IF EXISTS employee`
	_, err = conn.Exec(context.Background(), dropStmt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exec for drop table failed: %v\n", err)
	}

	var createStmt = `CREATE TABLE employee (id int PRIMARY KEY,
                                             name varchar,
                                             age int,
                                             language varchar)`
	_, err = conn.Exec(context.Background(), createStmt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exec for create table failed: %v\n", err)
	}
	fmt.Println("Created table employee")

	var insertStmt string = "INSERT INTO employee(id, name, age, language)" +
		" VALUES (1, 'John', 35, 'Go')"
	_, err = conn.Exec(context.Background(), insertStmt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exec for create table failed: %v\n", err)
	}
	// fmt.Printf("Inserted data: %s\n", insertStmt)

	// Read from the table.
	var name, language string
	var age int
	rows, err := conn.Query(context.Background(), "SELECT name, age, language FROM employee WHERE id = 1")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&name, &age, &language)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	printHostLoad()
}

func executeQueriesOnPool() {
	fmt.Printf("Acquiring %d connections from pool ...\n", numGoRoutines)
	for i := 0; i < numGoRoutines; i++ {
		wg.Add(1)
		go executeQueryOnPool("GO Routine " + strconv.Itoa(i))
	}
	time.Sleep(1 * time.Second)
	printHostLoad()
	wg.Wait()
}

func executeQueryOnPool(grid string) {
	defer wg.Done()
	for {
		// Read from the table.
		var name, language string
		var age int
		rows, err := pool.Query(context.Background(), "SELECT name, age, language FROM employee WHERE id = 1")
		if err != nil {
			log.Fatalf("pool.Query() failed, %s", err)
		}
		defer rows.Close()
		fstr := fmt.Sprintf("[%s] Query for id=1 returned: ", grid)
		for rows.Next() {
			err := rows.Scan(&name, &age, &language)
			if err != nil {
				log.Fatalf("rows.Scan() failed, %s", err)
			}
			fstr = fstr + fmt.Sprintf(" Row[%s, %d, %s] ", name, age, language)
		}
		err = rows.Err()
		if err != nil {
			fmt.Printf("%s, retrying ...\n", err)
			continue
		}
		time.Sleep(5 * time.Second)
		break
	}
}
