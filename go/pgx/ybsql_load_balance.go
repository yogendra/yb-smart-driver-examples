package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/yugabyte/pgx/v4"
)

// "github.com/jackc/pgx/v4/pgxpool"
// host     = "127.0.0.1,127.0.0.2"

const (
	host     = "127.0.0.1"
	port     = 5433
	user     = "yugabyte"
	password = "yugabyte"
	dbname   = "yugabyte"
)

var ybInstall string
var wg sync.WaitGroup

func main() {
	baseUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?refresh_interval=0",
		user, password, host, port, dbname)
	var interactive bool = false

	if len(os.Args) > 3 || len(os.Args) < 2 {
		log.Printf("Usage: ./ybsql_load_balance [-i] <path-to-ybdb-installation-dir>")
		log.Fatalf("Incorrect arguments: %s", os.Args)
	}
	if len(os.Args) == 2 {
		interactive = false
		ybInstall = os.Args[1]
	}
	if len(os.Args) == 3 {
		if os.Args[1] == "-i" {
			interactive = true
			ybInstall = os.Args[2]
		} else {
			if os.Args[2] == "-i" {
				interactive = true
				ybInstall = os.Args[1]
			} else {
				log.Printf("Usage: ./ybsql_load_balance [-i] <path-to-ybdb-installation-dir>")
				log.Fatalf("Incorrect arguments: %s", os.Args)
			}
		}
	}
	log.Printf("Received YBDB install path: %s", ybInstall)

	log.Print("Destroying earlier YBDB cluster, if any ...")
	err := exec.Command(ybInstall+"/bin/yb-ctl", "stop").Run()
	if err != nil {
		log.Fatalf("Could not stop earlier YBDB cluster: %s", err)
	}
	err = exec.Command(ybInstall+"/bin/yb-ctl", "destroy").Run()
	if err != nil {
		log.Fatalf("Could not destroy earlier YBDB cluster: %s", err)
	}

	log.Print("Starting a YBDB cluster with rf=3 ...")
	cmd := exec.Command(ybInstall+"/bin/yb-ctl", "create", "--rf", "3")
	var errout bytes.Buffer
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not start YBDB cluster: %s", errout.String())
	}
	defer exec.Command(ybInstall+"/bin/yb-ctl", "destroy").Run()
	time.Sleep(1 * time.Second)
	log.Printf("Started the cluster!")

	// Create a table and insert a row
	url := fmt.Sprintf("%s&load_balance=true", baseUrl)
	createTable(url, "---- Creating table ...")

	// make connections using the url via different go routines and check load balance
	performConcurrentReads(false, url, "---- Querying all servers ...")
	pause(interactive)

	// add a server with a different placement zone
	cmd = exec.Command(ybInstall+"/bin/yb-ctl", "add_node", "--placement_info", "cloud1.datacenter1.rack2")
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not add a YBDB server: %s", errout)
	}
	time.Sleep(5 * time.Second)
	performConcurrentReads(false, url, "---- Querying all servers after adding one more server ...")
	pause(interactive)

	// bring down a server and create new connections via go routines. check load balance
	log.Print("---- Stopping server 2 ...")
	cmd = exec.Command(ybInstall+"/bin/yb-ctl", "stop_node", "2")
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not stop a YBDB server: %s", errout)
	}
	performConcurrentReads(false, url, "---- Querying all servers after stopping server 2 ...")
	pause(interactive)

	// create new connections via go routines to new placement zone and check load balance
	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter1.rack2", baseUrl)
	performConcurrentReads(true, url, "---- Querying all servers in rack2 ...")
	verifyLoad(map[string]int{"127.0.0.1": 0, "127.0.0.2": 0, "127.0.0.3": 0, "127.0.0.4": 6})
	pause(interactive)

	// create new connections to both the zones and check load balance
	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter1.rack1,cloud1.datacenter1.rack2", baseUrl)
	performConcurrentReads(true, url, "---- Querying all servers in rack1 and rack2 ...")
	verifyLoad(map[string]int{"127.0.0.1": 3, "127.0.0.2": 0, "127.0.0.3": 3, "127.0.0.4": 6})

	log.Print("You can verify the connection count on http://127.0.0.1:13000/rpcz and others.")
	pause(interactive)
	log.Println("Closing the application")
}

func verifyLoad(expected map[string]int) {
	actual := pgx.GetHostLoad()["127.0.0.1"]
	for h, c := range expected {
		if actual[h] != c {
			log.Fatalf("For %s, expected count: %d, actual: %d", h, c, actual[h])
		}
	}
}

func pause(interactive bool) {
	if interactive {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Hit Enter to proceed: ")
		reader.ReadString('\n')
	}
}
func createTable(url string, msg string) {
	log.Print(msg)
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	pgx.PrintHostLoad()

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
	log.Println("Created table employee")

	var insertStmt string = "INSERT INTO employee(id, name, age, language)" +
		" VALUES (1, 'John', 35, 'Go')"
	_, err = conn.Exec(context.Background(), insertStmt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exec for create table failed: %v\n", err)
	}
	log.Printf("Inserted data: %s\n", insertStmt)

	// Read from the table.
	var name, language string
	var age int
	rows, err := conn.Query(context.Background(), "SELECT name, age, language FROM employee WHERE id = 1")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	// log.Printf("Query for id=1 returned: ")
	for rows.Next() {
		err := rows.Scan(&name, &age, &language)
		if err != nil {
			log.Fatal(err)
		}
		// log.Printf("Row[%s, %d, %s]\n", name, age, language)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}

func performConcurrentReads(keepConn bool, url string, msg string) {
	fmt.Println("\n", msg)
	pgx.PrintHostLoad()
	log.Println("Creating six connections across different Go routines")
	for i := 0; i <= 5; i++ {
		wg.Add(1)
		go queryTable(keepConn, "GO Routine "+strconv.Itoa(i), url)
	}
	time.Sleep(1 * time.Second)
	pgx.PrintHostLoad()
	wg.Wait()
}

func queryTable(keepConn bool, grid string, url string) {
	defer wg.Done()
	// log.Printf("[%s] Getting a connection ...", grid)
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Unable to connect to database: %v\n", grid, err)
		os.Exit(1)
	}
	if !keepConn {
		defer conn.Close(context.Background())
	}

	// Read from the table.
	var name, language string
	var age int
	// log.Printf("[%s] Executing select ...", grid)
	rows, err := conn.Query(context.Background(), "SELECT name, age, language FROM employee WHERE id = 1")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fstr := fmt.Sprintf("[%s] Query for id=1 returned: ", grid)
	for rows.Next() {
		err := rows.Scan(&name, &age, &language)
		if err != nil {
			log.Fatal(err)
		}
		fstr = fstr + fmt.Sprintf(" Row[%s, %d, %s] ", name, age, language)
	}
	// log.Println(fstr)
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(5 * time.Second)
}
