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
	"strings"
	"time"

	"github.com/yugabyte/pgx/v4"
)

const (
	host     = "127.0.0.1"
	port     = 5433
	user     = "yugabyte"
	password = "yugabyte"
	dbname   = "yugabyte"
	numconns = 84
)

var ybInstallPath string
var connCloseChan chan int = make(chan int)
var baseUrl string = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?yb_servers_refresh_interval=0",
	user, password, host, port, dbname)
var baseLocalhostUrl = fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?yb_servers_refresh_interval=0",
	user, password, port, dbname)
var interactive bool = false
var usePool bool = false
var errnil error = nil

func main() {
	if len(os.Args) > 4 || len(os.Args) < 2 {
		fmt.Println("Usage: ./ybsql_load_balance [-i] [--pool] <path-to-ybdb-installation-dir>")
		fmt.Printf("Incorrect arguments: %s\n", os.Args)
		os.Exit(1)
	}
	args := os.Args[1:]
	for _, a := range args {
		switch a {
		case "-i":
			interactive = true
		case "--pool":
			usePool = true
		default:
			_, err := os.Stat(a)
			if err != nil && os.IsNotExist(err) {
				fmt.Printf("Path does not exist/Invalid argument: %s\n", a)
				os.Exit(1)
			}
			ybInstallPath = a
		}
	}
	fmt.Printf("Received YBDB install path: %s\n", ybInstallPath)

	fmt.Println("Destroying earlier YBDB cluster, if any ...")
	err := exec.Command(ybInstallPath+"/bin/yb-ctl", "stop").Run()
	if err != nil {
		fmt.Printf("Could not stop earlier YBDB cluster: %s\n", err)
		os.Exit(1)
	}
	err = exec.Command(ybInstallPath+"/bin/yb-ctl", "destroy").Run()
	if err != nil {
		fmt.Printf("Could not destroy earlier YBDB cluster: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting a YBDB cluster with rf=3 ...")
	cmd := exec.Command(ybInstallPath+"/bin/yb-ctl", "create", "--rf", "3")
	var errout bytes.Buffer
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Could not start YBDB cluster: %s\n", errout.String())
		os.Exit(1)
	}
	defer exec.Command(ybInstallPath+"/bin/yb-ctl", "destroy").Run()
	time.Sleep(1 * time.Second)
	fmt.Println("Started the cluster!")

	if usePool {
		startPoolExample()
	} else {
		startExample()
	}
}

func startExample() {
	// Create a table and insert a row
	url := fmt.Sprintf("%s&load_balance=true", baseUrl)
	fmt.Printf("Using connection url: %s\n", url)
	pause()
	createTable(url)
	verifyZoneList(map[string]map[string][]string{host: {"cloud1.datacenter1.rack1": {"127.0.0.1", "127.0.0.2", "127.0.0.3"}}})
	printAZInfo()
	pause()

	// make connections using the url via different go routines and check load balance
	executeQueries(url, errnil)
	if interactive {
		fmt.Println("You can verify the connection counts on http://127.0.0.1:13000/rpcz and similar urls for other servers.")
	}
	pause()
	closeConns(numconns)

	// add a server with a different placement zone
	fmt.Println("Adding a new server in zone datacenter1 rack2...")
	var errout bytes.Buffer
	cmd := exec.Command(ybInstallPath+"/bin/yb-ctl", "add_node", "--placement_info", "cloud1.datacenter1.rack2")
	cmd.Stderr = &errout
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Could not add a YBDB server: %s", errout)
	}
	time.Sleep(5 * time.Second)

	fmt.Println("Adding a new server in zone datacenter1 rack3...")
	var errout1 bytes.Buffer
	cmd1 := exec.Command(ybInstallPath+"/bin/yb-ctl", "add_node", "--placement_info", "cloud1.datacenter1.rack3")
	cmd1.Stderr = &errout1
	err1 := cmd1.Run()
	if err1 != nil {
		log.Fatalf("Could not add a YBDB server: %s", errout)
	}
	time.Sleep(5 * time.Second)

	fmt.Println("Adding a new server in zone datacenter2 rack1...")
	var errout2 bytes.Buffer
	cmd2 := exec.Command(ybInstallPath+"/bin/yb-ctl", "add_node", "--placement_info", "cloud1.datacenter2.rack1")
	cmd2.Stderr = &errout2
	err2 := cmd2.Run()
	if err2 != nil {
		log.Fatalf("Could not add a YBDB server: %s", errout)
	}
	time.Sleep(5 * time.Second)

	fmt.Println("Adding a new server in zone datacenter2 rack2...")
	var errout3 bytes.Buffer
	cmd3 := exec.Command(ybInstallPath+"/bin/yb-ctl", "add_node", "--placement_info", "cloud1.datacenter2.rack2")
	cmd3.Stderr = &errout3
	err3 := cmd3.Run()
	if err3 != nil {
		log.Fatalf("Could not add a YBDB server: %s", errout)
	}
	time.Sleep(5 * time.Second)

	fmt.Println("Adding a new server in zone datacenter2 rack3...")
	var errout4 bytes.Buffer
	cmd4 := exec.Command(ybInstallPath+"/bin/yb-ctl", "add_node", "--placement_info", "cloud1.datacenter2.rack3")
	cmd4.Stderr = &errout4
	err4 := cmd4.Run()
	if err4 != nil {
		log.Fatalf("Could not add a YBDB server: %s", errout)
	}
	time.Sleep(5 * time.Second)

	executeQueries(url, errnil)
	verifyZoneList(map[string]map[string][]string{host: {"cloud1.datacenter1.rack1": {"127.0.0.1", "127.0.0.2", "127.0.0.3"},
		"cloud1.datacenter1.rack2": {"127.0.0.4"}, "cloud1.datacenter1.rack3": {"127.0.0.5"}, "cloud1.datacenter2.rack1": {"127.0.0.6"}, "cloud1.datacenter2.rack2": {"127.0.0.7"}, "cloud1.datacenter2.rack3": {"127.0.0.8"}}})
	printAZInfo()
	pause()
	closeConns(numconns)

	// create new connections via go routines to new placement zone and check load balance
	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter1.rack2", baseUrl)
	fmt.Printf("Using connection url:\n    %s\n", url)
	executeQueries(url, errnil)
	verifyLoad(map[string]int{"127.0.0.1": 0, "127.0.0.2": 0, "127.0.0.3": 0, "127.0.0.4": numconns, "127.0.0.5": 0, "127.0.0.6": 0, "127.0.0.7": 0, "127.0.0.8": 0})
	pause()
	closeConns(numconns)

	// bring down a server and create new connections via go routines. check load balance
	fmt.Println("Stopping server 4 ...")
	cmd = exec.Command(ybInstallPath+"/bin/yb-ctl", "stop_node", "4")
	cmd.Stderr = &errout
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not stop the YBDB server: %s", errout)
	}

	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter1.rack2", baseUrl)
	executeQueries(url, errnil)
	connCnt := numconns / 7
	verifyLoad(map[string]int{"127.0.0.1": connCnt, "127.0.0.2": connCnt, "127.0.0.3": connCnt, "127.0.0.4": 0, "127.0.0.5": connCnt, "127.0.0.6": connCnt, "127.0.0.7": connCnt, "127.0.0.8": connCnt})
	pause()
	closeConns(numconns)

	// create new connections to both the zones and check load balance
	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter2.rack1:2,cloud1.datacenter1.rack2:1", baseUrl)
	fmt.Printf("Using connection url:\n    %s\n", url)
	executeQueries(url, errnil)
	verifyLoad(map[string]int{"127.0.0.1": 0, "127.0.0.2": 0, "127.0.0.3": 0, "127.0.0.4": 0, "127.0.0.5": 0, "127.0.0.6": numconns, "127.0.0.7": 0, "127.0.0.8": 0})
	pause()
	closeConns(numconns)

	// create new connections to both the zones and check load balance
	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter2.*:2,cloud1.datacenter1.rack2:1", baseUrl)
	fmt.Printf("Using connection url:\n    %s\n", url)
	executeQueries(url, errnil)
	verifyLoad(map[string]int{"127.0.0.1": 0, "127.0.0.2": 0, "127.0.0.3": 0, "127.0.0.4": 0, "127.0.0.5": 0, "127.0.0.6": numconns / 3, "127.0.0.7": numconns / 3, "127.0.0.8": numconns / 3})
	pause()
	closeConns(numconns)

	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter2.*:2,cloud1.datacenter1.*:1", baseUrl)
	fmt.Printf("Using connection url:\n    %s\n", url)
	executeQueries(url, errnil)
	verifyLoad(map[string]int{"127.0.0.1": numconns / 4, "127.0.0.2": numconns / 4, "127.0.0.3": numconns / 4, "127.0.0.4": 0, "127.0.0.5": numconns / 4, "127.0.0.6": 0, "127.0.0.7": 0, "127.0.0.8": 0})
	pause()
	closeConns(numconns)

	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter2.rack3:1,cloud1.datacenter1.rack3:1,cloud1.datacenter1.rack1:2,cloud1.datacenter2.rack2:2", baseUrl)
	fmt.Printf("Using connection url:\n    %s\n", url)
	executeQueries(url, errnil)
	verifyLoad(map[string]int{"127.0.0.1": 0, "127.0.0.2": 0, "127.0.0.3": 0, "127.0.0.4": 0, "127.0.0.5": numconns / 2, "127.0.0.6": 0, "127.0.0.7": 0, "127.0.0.8": numconns / 2})
	pause()
	closeConns(numconns)

	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter2.rack3:,cloud1.datacenter1.rack3:1,cloud1.datacenter1.rack1:2,cloud1.datacenter2.rack2:2", baseUrl)
	fmt.Printf("Using connection url:\n    %s\n", url)
	zones := strings.Split("cloud1.datacenter2.rack3:", ":")
	str := "Invalid preference value for " + zones[0] + ": " + zones[1]
	errnew := fmt.Errorf(str)
	executeQueries(url, errnew)
	fmt.Println("Error :", errnew)
	pause()

	url = fmt.Sprintf("%s&load_balance=true&topology_keys=cloud1.datacenter2.rack3:1,cloud1.datacenter1.rack3:1,cloud1.datacenter1.rack1:a,cloud1.datacenter2.rack2:2", baseUrl)
	fmt.Printf("Using connection url:\n    %s\n", url)
	zones = strings.Split("cloud1.datacenter1.rack1:a", ":")
	str = "Invalid preference value for " + zones[0] + ": " + zones[1]
	errnew = fmt.Errorf(str)
	executeQueries(url, errnew)
	fmt.Println("Error :", errnew)
	pause()

	fmt.Println("Closing the application ...")
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

func pause() {
	if interactive {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nPress Enter/return to proceed: ")
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
	// fmt.Printf("Inserted data: %s\n", insertStmt)

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

func executeQueries(url string, err1 error) {
	fmt.Printf("Creating %d connections ...\n", numconns)
	for i := 0; i < numconns; i++ {
		go executeQuery("GO Routine "+strconv.Itoa(i), url, connCloseChan, err1)
	}
	time.Sleep(5 * time.Second)
	if err1 == nil {
		printHostLoad()
	}
}

func executeQuery(grid string, url string, ccChan chan int, err1 error) {
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		if err.Error() != err1.Error() {
			fmt.Fprintf(os.Stderr, "[%s] Unable to connect to database: %v\n", grid, err)
			os.Exit(1)
		} else {
			return
		}
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
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	// log.Println(fstr)
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
		str := "Placement info details of cluster (" + k + "): "
		for z, hosts := range zl {
			str = str + fmt.Sprintf("\n    AZ [%s]: ", z)
			for _, s := range hosts {
				str = str + fmt.Sprintf("%s, ", s)
			}
		}
		fmt.Println(str)
	}
}
