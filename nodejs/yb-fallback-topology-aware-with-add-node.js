const { spawn } = require("child_process");
const pg = require('@yugabytedb/pg');
const process = require('process')
var assert = require('assert');

const yb_path = process.env.YB_PATH;

async function createConnection(){
    const yburl = "postgresql://yugabyte:yugabyte@127.0.0.1:5433/yugabyte?loadBalance=true&&topologyKeys=cloud1.datacenter2.rack1&&ybServersRefreshInterval=0"
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
            const createCluster = spawn("./bin/yb-ctl", ["create", "--rf", "7", "--placement_info", "cloud1.datacenter1.rack1,cloud1.datacenter1.rack1,cloud1.datacenter1.rack2,cloud1.datacenter1.rack3,cloud1.datacenter2.rack1,cloud1.datacenter2.rack2,cloud1.datacenter2.rack3"]);
            console.log("Creating cluster with RF 7 with 6 different placement infos..")
            createCluster.on('close', async (code) => {
                if(code === 0){
                    let clientArray = []
                    let numConnections = 16
                    let timeToMakeConnections = numConnections * 200;
                    let timeToEndConnections = numConnections * 50;
                    console.log("Creating",numConnections, "connections.");
                    clientArray = await createNumConnections(numConnections)
                
                    setTimeout(async () => {
                        console.log(numConnections, "connections are created!");
                        let connectionMap = pg.Client.connectionMap;
                        console.log("Connection Map: ", connectionMap)
                        const hosts = connectionMap.keys();
                        for(let value of hosts){
                            let cnt = connectionMap.get(value);
                            if(value === '127.0.0.5'){
                                assert.equal(cnt, 16, 'Node '+ value + ' is not balanced');
                            }else {
                                 assert.equal(cnt, 0, 'Node '+ value + ' is not balanced');
                            }
                        }
                        console.log("Nodes are all load Balanced on only one nodes matching with placement info mentioned in topology key.")

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

                            const addOneNode = spawn("./bin/yb-ctl", ["add_node" , "--placement_info", "cloud1.datacenter2.rack1"])
                            addOneNode.stdout.on("data", () => {
                                console.log("Adding one node with same placement info as we are specifying in topology key ... ")
                            })
                            addOneNode.on('close', async (code) => {
                               if(code === 0){
                                setTimeout(async () => {
                                    pg.Client.doHardRefresh = true;   
                                    clientArray = await createNumConnections(numConnections)
                                    setTimeout(() => {
                                        console.log(numConnections, "connections are created after adding one node.");
                                        let connectionMap = pg.Client.connectionMap;
                                        console.log("Connection Map: ", connectionMap)
                                        const hosts = connectionMap.keys();
                                        for(let value of hosts){
                                            let cnt = connectionMap.get(value);
                                            if(value === '127.0.0.5' || value === '127.0.0.8'){
                                                assert.equal(cnt, 8, 'Node '+ value + ' is not balanced');
                                            }else {
                                                 assert.equal(cnt, 0, 'Node '+ value + ' is not balanced');
                                            }
                                        }
                                        console.log("Nodes are all load Balanced across two nodes after adding node with same placement info.")
                                    }, timeToMakeConnections)
                                }, 1000)
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