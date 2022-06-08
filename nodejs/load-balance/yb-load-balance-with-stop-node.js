const { spawn } = require("child_process");
const pg = require('../../../node-postgres/packages/pg');
const process = require('process')
var assert = require('assert');

const yb_path = process.env.YB_PATH;

async function createConnection(){
    const yburl = "postgresql://yugabyte:yugabyte@localhost:5433/yugabyte?loadBalance=true"
    let client = new pg.Client(yburl);
    client.on('error', () => {
        // ignore the error and handle exiting 
    })
    await client.connect()
    client.connection.on('error', () => {
        // ignore the error and handle exiting 
    })
    return client;
}

async function createNumConnections(numConnections) {
    let clientArray = []
    for (let i=0; i<numConnections; i++) {
        if(i&1){
             clientArray.push(await createConnection())
        }else  {
            setTimeout(async() => {
                clientArray.push(await createConnection())
            }, 1000)
        }
    }
    return clientArray
}

async function endNumConnections(numConnections, clientArray){
    for (let i=0; i<numConnections; i++) {
        await clientArray[i].end((err) => {
            if(err){}
            else {
            }
        })
    }
}

function example(){
    process.chdir(yb_path)
    const destroyCluster = spawn("./bin/yb-ctl", ["destroy"]);
    destroyCluster.stdout.on('data', ()=>{
        console.log("Destroying earlier cluster, if any..")
    })
    destroyCluster.on('close', async (code) => {
        if(code === 0){
            const createCluster = spawn("./bin/yb-ctl", ["create", "--rf", "3"]);
            console.log("Creating cluster with RF 3 ..")
            createCluster.on('close', async (code) => {
                if(code === 0){
        
                    let clientArray = []
                    let numConnections = 12
                    let timeToMakeConnections = numConnections * 200;
                    let timeToEndConnections = numConnections * 50;
                    console.log("Creating",numConnections, "connections with load balance");
                    clientArray = await createNumConnections(numConnections)
                
                    setTimeout(async () => {
                        console.log(numConnections, "connections are created!");
                        let connectionMap = pg.Client.connectionMap;
                        console.log("Connection Map: ", connectionMap)
                        const hosts = connectionMap.keys();
                        for(let value of hosts){
                            let cnt = connectionMap.get(value);
                            assert.equal(cnt, 4, 'Node '+ value + ' is not balanced');
                        }
                        console.log("Nodes are all load Balanced!")

                        await endNumConnections(numConnections, clientArray);
                        setTimeout(async() => {
                            let connectionMap = pg.Client.connectionMap;
                            console.log("Connection Map: ", connectionMap)
                            const hosts = connectionMap.keys();
                            for(let value of hosts){
                                let cnt = connectionMap.get(value);
                                assert.equal(cnt, 0, 'All connections are not closed at this node: ' + value);
                            }
                            console.log("All connections are closed!")

                            const stopOneNode = spawn("./bin/yb-ctl", ["stop_node", "1"])
                            stopOneNode.stdout.on("data", () => {
                                console.log("Stopping node with host IP - 127.0.0.1")
                            })
                            stopOneNode.on('close', async (code) => {
                               if(code === 0){
                                clientArray = await createNumConnections(numConnections)
                                setTimeout(() => {
                                    console.log(numConnections, "connections are created after stopping node 1.");
                                    let connectionMap = pg.Client.connectionMap;
                                    console.log("Connection Map: ", connectionMap)
                                    const hosts = connectionMap.keys();
                                    for(let value of hosts){
                                        let cnt = connectionMap.get(value);
                                        assert.equal(cnt, 6, 'Node '+ value + 'is not balanced');
                                    }
                                    console.log("Nodes are all load Balanced after stopping node 1.")
                                }, timeToMakeConnections)
                               }
                            })
                        },timeToEndConnections)
                    }, timeToMakeConnections)
                
                    
                }
            })
        }
    })
}


example()