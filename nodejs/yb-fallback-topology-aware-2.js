const { spawn } = require("child_process");
const process = require('process')
var assert = require('assert');
const pg = require('@yugabytedb/pg');
const yb_path = process.env.YB_PATH;

//Testing fallback to second preference zone when all servers in primary preference zone are down.
async function createConnection() {
    const yburl = "postgresql://yugabyte:yugabyte@127.0.0.1:5433/yugabyte?loadBalance=true&topologyKeys=cloud1.datacenter1.rack1:2,cloud1.datacenter2.rack1:1,cloud1.datacenter1.rack2:3"
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
    for (let i = 0; i < numConnections; i++) {
        if (i & 1) {
            clientArray.push(await createConnection())
        } else {
            setTimeout(async () => {
                clientArray.push(await createConnection())
            }, 1000)
        }
    }
    return clientArray
}

async function endNumConnections(numConnections, clientArray) {
    for (let i = 0; i < numConnections; i++) {
        await clientArray[i].end((err) => {
            if (err) {
                console.log("Error in closing connection", err)
            }
        })
    }
}

function example() {
    process.chdir(yb_path)
    const destroyCluster = spawn("./bin/yb-ctl", ["destroy"]);
    destroyCluster.stdout.on('data', () => {
        console.log("Destroying earlier cluster, if any..")
    })
    destroyCluster.on('close', async (code) => {
        if (code === 0) {
            const createCluster = spawn("./bin/yb-ctl", ["create", "--rf", "3", "--placement_info", "cloud1.datacenter1.rack1,cloud1.datacenter1.rack2,cloud1.datacenter2.rack1"]);
            console.log("Creating cluster with RF 3 with 3 different placement infos..")
            createCluster.on('close', async (code) => {
                if (code === 0) {
                    let clientArray = []
                    let numConnections = 12
                    let timeToMakeConnections = numConnections * 200;
                    let timeToEndConnections = numConnections * 50;
                    console.log("Testing fallback to second preference zone when all servers in primary preference zone are down.")
                    console.log("Creating", numConnections, "connections.");
                    clientArray = await createNumConnections(numConnections)

                    setTimeout(async () => {
                        console.log(numConnections, "connections are created!");
                        let connectionMap = pg.Client.connectionMap;
                        console.log("Connection Map: ", connectionMap)
                        const hosts = connectionMap.keys();
                        for (let value of hosts) {
                            let cnt = connectionMap.get(value);
                            if (value === '127.0.0.3') {
                                assert.equal(cnt, 12, 'Node ' + value + ' is not balanced');
                            } else {
                                assert.equal(cnt, 0, 'Node ' + value + ' is not balanced');
                            }
                        }
                        console.log("Nodes are all load Balanced on only one nodes matching with placement info mentioned in topology key with preference value 1.")

                        await endNumConnections(numConnections, clientArray);
                        setTimeout(async () => {
                            let connectionMap = pg.Client.connectionMap;
                            console.log("Connection Map: ", connectionMap)
                            const hosts = connectionMap.keys();
                            for (let value of hosts) {
                                let cnt = connectionMap.get(value);
                                assert.equal(cnt, 0, 'All connections are not closed at this node: ' + value);
                            }
                            console.log("All connections are closed!")

                            const stopOneNode = spawn("./bin/yb-ctl", ["stop_node", "3"])
                            stopOneNode.stdout.on("data", () => {
                                console.log("Stopping node with host IP - 127.0.0.3")
                            })
                            stopOneNode.on('close', async (code) => {
                                if (code === 0) {
                                    clientArray = await createNumConnections(numConnections)
                                    setTimeout(() => {
                                        console.log(numConnections, "connections are created after stopping node 3.");
                                        let connectionMap = pg.Client.connectionMap;
                                        console.log("Connection Map: ", connectionMap)
                                        const hosts = connectionMap.keys();
                                        for (let value of hosts) {
                                            let cnt = connectionMap.get(value);
                                            if (value === '127.0.0.1') {
                                                assert.equal(cnt, 12, 'Node ' + value + ' is not balanced');
                                            } else {
                                                assert.equal(cnt, 0, 'Node ' + value + ' is not balanced');
                                            }
                                        }
                                        console.log("Since all server of preference value 1 are down load goes to the server with preference value 2 only, no load on servers with preference value 3.")
                                    }, timeToMakeConnections)
                                }
                            })
                        }, timeToEndConnections)
                    }, timeToMakeConnections)
                }
            })
        }
    })
}

example()