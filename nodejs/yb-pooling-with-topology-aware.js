const { spawn } = require("child_process");
const pg = require('../../node-postgres/packages/pg/lib');
const Pool = require('../../node-postgres/packages/pg-pool')
const process = require('process')
var assert = require('assert');

const yb_path = process.env.YB_PATH;

function createPool(){
    let pool = new Pool({
        user: 'yugabyte',
        password: 'yugabyte',
        host: 'localhost',
        port: 5433,
        loadBalance: true,
        database: 'yugabyte',
        max: 100,
        topologyKeys:'cloud1.datacenter1.rack1'
    })
    return pool
}

async function createNumConnections(numConnections, pool) {
    for(let i=0;i<numConnections;i++){
        if(i&1){
            await pool.connect();
        }else{
            setTimeout(async () => {
                await pool.connect();
            },2000)
        }
    }
}

function endNumConnections(numConnections, pool){
    for (let i=0; i<numConnections; i++) {
        pool._clients[i].on('error',()=>{})
        pool._clients[i].connection.on('error',()=>{})
    }
    for (let i=0; i<numConnections; i++) {
        pool._clients[i].end();
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
            const createCluster = spawn("./bin/yb-ctl", ["create", "--rf", "3", "--placement_info", "cloud1.datacenter1.rack1,cloud1.datacenter1.rack1,cloud1.datacenter1.rack2"]);
            console.log("Creating cluster with RF 3 with two different placement infos..")
            createCluster.on('close', async (code) => {
                if(code === 0){

                    let numConnections = 12
                    let timeToMakeConnections = numConnections * 200;
                    let timeToEndConnections = numConnections * 50;
                    console.log("Creating pool of max 100 connections..")
                    let pool = createPool();
                    console.log("Creating", numConnections, "connections  out of pool with topology key which matches with two nodes in the cluster...");
                    await createNumConnections(numConnections, pool)
                
                    setTimeout(async () => {
                        console.log(numConnections, "connections are created!");
                        let connectionMap = pool.Client.connectionMap;
                        console.log("Connection Map: ", connectionMap)
                        const hosts = connectionMap.keys();
                        for(let value of hosts){
                            let cnt = connectionMap.get(value);
                            if(value === '127.0.0.3'){
                                assert.equal(cnt, 0, 'Node '+ value + ' is not balanced');
                            }else {
                                 assert.equal(cnt, 6, 'Node '+ value + ' is not balanced');
                            }
                        }
                        console.log("Nodes are all load Balanced on only two nodes matching with placement info mentioned in topology key.")

                        endNumConnections(numConnections, pool);
                        setTimeout(async() => {
                            let connectionMap = pool.Client.connectionMap;
                            console.log("Connection Map: ", connectionMap)
                            const hosts = connectionMap.keys();
                            for(let value of hosts){
                                let cnt = connectionMap.get(value);
                                assert.equal(cnt, 0, 'All connections are not closed at this node: ' + value);
                            }
                            console.log("All connections are closed!")

                            const addOneNode = spawn("./bin/yb-ctl", ["add_node", "--placement_info", "cloud1.datacenter1.rack1"])
                            addOneNode.stdout.on("data", () => {
                                console.log("Adding one node with same placement info as we are specifying in topology key ... ")
                            })
                            addOneNode.on('close', async (code) => {
                               if(code === 0){
                                setTimeout(async () => {
                                    pg.Client.doHardRefresh = true;   
                                    await createNumConnections(numConnections, pool)
                                    setTimeout(() => {
                                        console.log(numConnections, "connections are created after adding one node.");
                                        let connectionMap = pool.Client.connectionMap;
                                        console.log("Connection Map: ", connectionMap)
                                        const hosts = connectionMap.keys();
                                        for(let value of hosts){
                                            let cnt = connectionMap.get(value);
                                            if(value === '127.0.0.3'){
                                                assert.equal(cnt, 0, 'Node '+ value + ' is not balanced');
                                            }else {
                                                 assert.equal(cnt, 4, 'Node '+ value + ' is not balanced');
                                            }
                                        }
                                        console.log("Nodes are all load Balanced across three nodes after adding node of same placement info.")
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