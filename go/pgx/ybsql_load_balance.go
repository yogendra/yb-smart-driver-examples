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

const (
	host     = "127.0.0.1"
	port     = 5433
	user     = "yugabyte"
	password = "yugabyte"
	dbname   = "yugabyte"
	numconns = 12
)

var ybInstall string
var wg sync.WaitGroup
var connCloseChan chan int = make(chan int)

func main() {
	baseUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?refresh_interval=0",
		user, password, host, port, dbname)
	var interactive bool = false

	if len(os.Args) > 3 || len(os.Args) < 2 {
		fmt.Println("Usage: ./ybsql_load_balance [-i] <path-to-ybdb-installation-dir>")
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
				fmt.Println("Usage: ./ybsql_load_balance [-i] <path-to-ybdb-installation-dir>")
				log.Fatalf("Incorrect arguments: %s", os.Args)
			}
		}
	}
	fmt.Printf("Received YBDB install path: %s\n", ybInstall)

	fmt.Println("Destroying earlier YBDB cluster, if any ...")
	err := exec.Command(ybInstall+"/bin/yb-ctl", "stop").Run()
	if err != nil {
		log.Fatalf("Could not stop earlier YBDB cluster: %s", err)
	}
	err = exec.Command(ybInstall+"/bin/yb-ctl", "destroy").Run()
	if err != nil {
		log.Fatalf("Could not destroy earlier YBDB cluster: %s", err)
	}

	fmt.Println("Starting a YBDB cluster with rf=3 ...")
	cmd := exec.Command(ybInstall+"/bin/yb-ctl", "create", "--rf", "3")
	var errout bytes.Buffer
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not start YBDB cluster: %s", errout.String())
	}
	defer exec.Command(ybInstall+"/bin/yb-ctl", "destroy").Run()
	time.Sleep(1 * time.Second)
	fmt.Println("Started the cluster!")

	// Create a table and insert a row
	url := fmt.Sprintf("%s&load_balance=true", baseUrl)
	createTable(url)
	verifyZoneList(map[string]map[string][]string{"127.0.0.1": {"cloud1.datacenter1.rack1": {"127.0.0.1", "127.0.0.2", "127.0.0.3"}}})
	printAZInfo()
	pause(interactive)

	// make connections using the url via different go routines and check load balance
	executeQueries(false, url, "---- Querying all servers ...")
	pause(interactive)
	closeConns(numconns)

	// add a server with a different placement zone
	fmt.Println("Adding a new server in zone rack2 ...")
	cmd = exec.Command(ybInstall+"/bin/yb-ctl", "add_node", "--placement_info", "cloud1.datacenter1.rack2")
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not add a YBDB server: %s", errout)
	}
	time.Sleep(5 * time.Second)
	executeQueries(false, url, "---- Querying all servers after adding one more server ...")
	verifyZoneList(map[string]map[string][]string{"127.0.0.1": {"cloud1.datacenter1.rack1": {"127.0.0.1", "127.0.0.2", "127.0.0.3"},
		"cloud1.datacenter1.rack2": {"127.0.0.4"}}})
	printAZInfo()
	pause(interactive)
	closeConns(numconns)

	// bring down a server and create new connections via go routines. check load balance
	fmt.Println("Stopping server 2 ...")
	cmd = exec.Command(ybInstall+"/bin/yb-ctl", "stop_node", "2")
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not stop the YBDB server: %s", errout)
	}
	executeQueries(true, url, "---- Querying all servers after stopping server 2 ...")
	connCnt := numconns / 3
	verifyLoad(map[string]int{"127.0.0.1": connCnt, "127.0.0.2": 0, "127.0.0.3": connCnt, "127.0.0.4": connCnt})
	if interactive {
		fmt.Println("You can verify the connection count on http://127.0.0.4:13000/rpcz and similar urls for other servers.")
	}
	pause(interactive)

	// create new connections via go routines to new placement zone and check load balance
	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter1.rack2", baseUrl)
	executeQueries(true, url, "---- Querying all servers in rack2 ...")
	verifyLoad(map[string]int{"127.0.0.1": connCnt, "127.0.0.2": 0, "127.0.0.3": connCnt, "127.0.0.4": connCnt + numconns})
	if interactive {
		fmt.Println("You can verify the connection count on http://127.0.0.4:13000/rpcz and similar urls for other servers.")
	}
	pause(interactive)

	// create new connections to both the zones and check load balance
	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter1.rack1,cloud1.datacenter1.rack2", baseUrl)
	executeQueries(true, url, "---- Querying all servers in rack1 and rack2 ...")
	verifyLoad(map[string]int{"127.0.0.1": connCnt + (numconns / 2), "127.0.0.2": 0, "127.0.0.3": connCnt + (numconns / 2), "127.0.0.4": connCnt + numconns})
	if interactive {
		fmt.Println("You can verify the connection count on http://127.0.0.1:13000/rpcz and similar urls for other servers.")
	}
	pause(interactive)
	closeConns(3 * numconns)
	fmt.Println("Closing the application")
}

func closeConns(num int) {
	fmt.Printf("Closing %d connections ...\n", num)
	for i := 0; i < num; i++ {
		connCloseChan <- i
	}
}

func verifyLoad(expected map[string]int) {
	actual := pgx.GetHostLoad()[host]
	for h, expectedCnt := range expected {
		if actual[h] != expectedCnt {
			log.Fatalf("For %s, expected count: %d, actual: %d", h, expectedCnt, actual[h])
		}
	}
}

func verifyZoneList(expected map[string]map[string][]string) {
	actual := pgx.GetAZInfo()
	if len(expected) != len(actual) {
		log.Fatalf("Found %d clusters, expected %d", len(actual), len(expected))
	}
	for cName, zList := range expected {
		zlActual, found := actual[cName]
		if !found {
			log.Fatalf("Cluster %s not found!", cName)
		}
		if len(zList) != len(zlActual) {
			log.Fatalf("Number of zones (%d) in cluster %s does not match with expected number (%d)", len(zlActual), cName, len(zList))
		}
		for z, list := range zList {
			hostsActual, found := actual[cName][z]
			if !found {
				log.Fatalf("Zone %s for cluster %s not found!", z, cName)
			}
			if len(list) != len(hostsActual) {
				log.Fatalf("Number of hosts (%d) in zone %s for cluster %s does not match with expected number (%d)", len(hostsActual), z, cName, len(list))
			}
			for _, h := range list {
				found := false
				for _, hActual := range hostsActual {
					if h == hActual {
						found = true
						continue
					}
				}
				if !found {
					log.Fatalf("Host %s not in zone %s for cluster %s", h, z, cName)
				}
			}
		}
	}
}

func pause(interactive bool) {
	if interactive {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Press Enter/return to proceed: ")
		reader.ReadString('\n')
	}
}

func createTable(url string) {
	fmt.Println("Creating table ...")
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

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
	fmt.Printf("Inserted data: %s\n", insertStmt)

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
	printHostLoad()
}

func executeQueries(keepConn bool, url string, msg string) {
	fmt.Println(msg)
	fmt.Printf("Creating %d connections across different Go routines\n", numconns)
	// if !keepConn {
	// 	fmt.Println("Connections will be closed when Go routines complete their task")
	// } else {
	// 	fmt.Println("Connections will be retained even after the Go routines complete their task")
	// }
	for i := 0; i < numconns; i++ {
		// wg.Add(1)
		go executeQuery("GO Routine "+strconv.Itoa(i), url, connCloseChan)
	}
	time.Sleep(5 * time.Second)
	printHostLoad()
	// wg.Wait()
}

func executeQuery(grid string, url string, ccChan chan int) {
	// defer wg.Done()
	// log.Printf("[%s] Getting a connection ...", grid)
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Unable to connect to database: %v\n", grid, err)
		os.Exit(1)
	}
	// if !keepConn {
	// 	defer conn.Close(context.Background())
	// }

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
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	// log.Println(fstr)
	// time.Sleep(5 * time.Second)
	_, ok := <-ccChan
	if ok {
		conn.Close(context.Background())
	}
}

func printHostLoad() {
	for k, cli := range pgx.GetHostLoad() {
		str := "Current load on cluster (" + k + "): "
		for h, c := range cli {
			str = str + fmt.Sprintf("\n%-30s:%5d", h, c)
		}
		fmt.Println(str)
	}
}

func printAZInfo() {
	for k, zl := range pgx.GetAZInfo() {
		str := "AZ details of cluster (" + k + "): "
		for z, hosts := range zl {
			str = str + fmt.Sprintf("\nAZ [%s]: ", z)
			for _, s := range hosts {
				str = str + fmt.Sprintf("%s, ", s)
			}
		}
		fmt.Println(str)
	}
}
