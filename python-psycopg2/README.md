# Setup

Set the environment variable `YB_PATH` to the path to your Yugabyte installed directory.
Install the psycopg2 smart driver (TODO : Mention steps after package publishing)

# Run the test
Use the following command:
```
python testname.py
```

Tests & examples regarding the Python driver with load_balancing feature will be available here.✅

# Uniform/Cluster Aware Test Cases
|Done|Name|Notes|File|
| - | - | - | - |
|-| Create one connection to the cluster and create a table. | Control connection is not persistent for python smart driver| - |
|✅| Create multiple connections (12+, in multiples of three) in a loop to the cluster using the same url/properties and perform a simple SELECT. | 4 connections for each node were found. |  test_uniformloadbalancer.py |
|✅| Create multiple connections (12+, in multiples of three) concurrently via threads using the same url/properties and perform a simple SELECT. | 4 connections each node were found. |  test_uniformloadbalancer.py |
|✅| Bring down the server s2 not passed in the url/properties and create (10) more connections from the application.                             | 1.Stop any node                                 -->Working fine; the connections were distributed between rest two nodes | test_uniformloadbalancer.py |
|✅| Add a new server host (s4) to the cluster and create (8) new connections to the cluster with the same url/properties.                        |  The connections were equally distributed among the 4 nodes. |  test_uniformloadbalancer.py |
|✅| Create a connection to the cluster using the hostname instead of IP (e.g. use “localhost” instead of 127.0.0.1) in the url/properties.       |  | test_uniformloadbalancer.py |
|✅| [Pool] Create a pool of connections to the cluster using available APIs and using the initial url/properties.                                | |test_uniformloadbalancer.py |
|| [Pool] Bring down a tserver not specified in the url/properties and increase the number of connections in the pool.                          | | |


# Topology Aware Test Cases
|Done|Name|Notes|File|
| - | - | - | - |
|-| Create one connection to the cluster and create a table.| | |
|✅| Create multiple connections (12+, in multiples of two) in a loop to the cluster using the same url/properties and perform a simple SELECT.   | | |
|✅| Create multiple connections (12+, in multiples of three) concurrently via threads using the same url/properties and perform a simple SELECT. | | |
|✅| Bring down server s2 and create (10) more connections from the application using the same url/properties.                                    | | |
|✅| Add a new tserver host (s4) with placement info the same as that of s1/s2, to the cluster and create (10+, in multiples of two) new connections to the cluster with the same url/properties.  | | |
|✅| Create a connection to the cluster using the hostname instead of IP (e.g. use “localhost” instead of 127.0.0.1) in the url/properties.       | | |
|✅| Create (10+, in multiples of two) new connections with url/properties specifying both the placements as comma-separated values of topology-keys.| | |
|✅| [Pool] Create a pool of connections to the cluster using available APIs and using the above url/properties.                                  | | |
|| [Pool] Bring down s4 and increase the number of connections in the pool.                                                                     | | |

# Miscellanous Test Cases

|Done|Name|Notes|File|
| - | - | - | - |
|✅| Start a yugabyte cluster with non-default port for YSQL `./bin/yugabyted start --ysql_port 5544`.Do not specify port in url and properties when connecting to the cluster| The application fails to connect to the cluster | |
|✅| Do not specify port in url and properties when connecting to the cluster. | The application still is able to connect with the cluster | |
|✅| Create 3 connections each using the accepted Connection URI formats (4 formats)  | 4 connections on each node were found. | |