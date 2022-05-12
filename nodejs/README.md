# Follow the steps to run this example - 
### 1. Getting the smart-driver locally installed 
- Clone the repository using:
```
git clone -b smart-driver-feature https://github.com/yugabyte/node-postgres.git
```
- Go to the `node-postgres` folder using:
```
cd node-postgres
```
- Install the node dependencies:
```
npm install 
```
### 2. To verify the Load Balance feature of the smart-driver 
- Get this example locally using:
    ```
    git clone -b nodejs-driver-example https://github.com/yugabyte/driver-examples.git
    ```
    and go to the nodejs examples folder:
    ```
    cd driver-examples/nodejs
    ```
- Now, in the `example.js` file change the path for smart-driver package with the relative path of your local smart-driver clone in the require function in the following line - 
```
const pg = require('./node-postgres/packages/pg');
```
- Get your YugabyteDB cluster up with replication factor 3 using the following command:
```
./bin/yb-ctl create --rf 3
```
- Before running example, verify your URL in the example your cluster configuration.:
    ```
    const yburl = "postgresql://<user>:<password>@<host>:<port>/<db>?load_balance=true";
    ```
- Run the example using: 
    ```
    node example.js
    ```
    1. You will get this output:
        ```
        Node connection counts after making connections: 
    
     		 Map(3) { '127.0.0.1' => 3, '127.0.0.2' => 3, '127.0.0.3' => 4 }
        ```
    2. You will then prompted with a quesiton to end connection, it will give this output after that pressing `enter`:
        ```
        Node connection counts after ending the connections: 
    
     		 Map(3) { '127.0.0.1' => 0, '127.0.0.2' => 0, '127.0.0.3' => 0 } 
        ```
    3. You will again prompted with a quesiton to stop a node and proceed next iteration for which first stop node using command:
        ```
        ./bin/yb-ctl stop_node 2
        ```
       after stopping node, press `enter` to create next set of connections which will give this as output:
       ```
       Node connection counts after some node is down and connections are made: 

 		 Map(2) { '127.0.0.1' => 5, '127.0.0.3' => 5 } 
       ```
         Note - The Map in each output shows the connection count of each node after creating/closing the connections.
### 3. To verify the Topology Aware Feature of smart-driver 
- Destroy the cluster using:
```
./bin/yb-ctl destroy
```
- Create new Cluster using placement info:
```
./bin/yb-ctl create --rf 3 --placement_info "cloud1.datacenter1.rack1,cloud1.datacenter1.rack1,cloud1.datacenter1.rack2"
```
- Now, change the URL of connection string in the example to use the `topology keys` by commenting line 6 and uncommenting line 7.
- Verfiy the URL given in the example your cluster configuration.
- Run the example using: 
    ```
    node example.js
    ```
    1. You will get this output:
        ```
       Node connection counts after making connections: 

 		 Map(3) { '127.0.0.2' => 5, '127.0.0.1' => 5, '127.0.0.3' => 0 } 
        ```
    2. You will then prompted with a quesiton to end connection, it will give this output after that pressing `enter`:
        ```
        Node connection counts after ending the connections: 
    
     		 Map(3) { '127.0.0.1' => 0, '127.0.0.2' => 0, '127.0.0.3' => 0 } 
        ```
    3. You will again prompted with a quesiton to stop a node and proceed next iteration for which first stop node using command:
        ```
        ./bin/yb-ctl stop_node 2
        ```
       after stopping node, press `enter` to create next set of connections which will give this as output:
       ```
       Node connection counts after some node is down and connections are made: 

 		 Map(2) { '127.0.0.1' => 10, '127.0.0.3' => 0 } 
       ```
       
Note - The Map in each output shows the connection count of each node after creating/closing the connections.  















