const { spawn } = require("child_process");
const pg = require('../../../node-postgres/packages/pg');
const Pool = require('../../../node-postgres/packages/pg-pool')
const process = require('process')
var assert = require('assert');

const yb_path = process.env.YB_PATH;

function createPool(){
    let pool = new Pool({
        user: 'yugabyte',
        password: 'yugabyte',
        host: 'localhost',
        port: 5433,
        load_balance: true,
        database: 'yugabyte',
        max: 100
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
            const createCluster = spawn("./bin/yb-ctl", ["create", "--rf", "3"]);
            console.log("Creating cluster with RF 3 ..")
            createCluster.on('close', async (code) => {
                if(code === 0){
        
                    let numConnections = 12
                    let timeToMakeConnections = numConnections * 200;
                    let timeToEndConnections = numConnections * 50;
                    console.log("Creating pool of max 100 connections..")
                    let pool = createPool();
                    console.log("Creating",numConnections, "connections with load_balance out of that pool..");
                    await createNumConnections(numConnections, pool)
                
                    setTimeout(async () => {
                        console.log(numConnections, "connections are created!");
                        let connectionMap = pool.Client.connectionMap;
                        console.log("Connection Map: ", connectionMap)
                        const hosts = connectionMap.keys();
                        for(let value of hosts){
                            let cnt = connectionMap.get(value);
                            assert.equal(cnt, 4, 'Node '+ value + ' is not balanced');
                        }
                        console.log("Nodes are all load Balanced!")

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

                            const stopOneNode = spawn("./bin/yb-ctl", ["stop_node", "1"])
                            stopOneNode.stdout.on("data", () => {
                                console.log("Stopping node with host IP - 127.0.0.1")
                            })
                            stopOneNode.on('close', async (code) => {
                               if(code === 0){
                                await createNumConnections(numConnections, pool)
                                setTimeout(() => {
                                    console.log(numConnections, "connections are created after stopping one node.");
                                    let connectionMap = pool.Client.connectionMap;
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