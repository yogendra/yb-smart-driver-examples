
const pg = require('./node-postgres/packages/pg');
const readline = require('readline');

async function createConnection(i){
    const yburl = "postgresql://yugabyte:yugabyte@127.0.0.1:5433/yugabyte?load_balance=true";
    // const yburl = "postgresql://yugabyte:yugabyte@localhost:5433/yugabyte?load_balance=true&&topology_keys=cloud1.datacenter1.rack1"
    let client = new pg.Client(yburl);
    client.on('error', () => {
        // ignore the error and handle exiting 
    })
    await client.connect((err) => {
        if(!err) {
            // console.log(i, 'connected')
        }
    })
    client.connection.on('error', () => {
        // ignore the error and handle exiting 
    })
    return client;
}

function askQuestion(query) {
    const rl = readline.createInterface({
        input: process.stdin,
        output: process.stdout,
    });

    return new Promise(resolve => rl.question(query, ans => {
        rl.close();
        resolve(ans);
    }))
}

async function createNumConnections(numConnections) {
    let clientArray = []
    for (let i=0; i<numConnections; i++) {
        if(i&1){
             clientArray.push(await createConnection(i))
        }else  {
            setTimeout(async() => {
                clientArray.push(await createConnection(i))
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
                // console.log(i, 'closed')
            }
        })
    }
}

(async () => {
    let clientArray = []
    let numConnections = 10
    clientArray = await createNumConnections(numConnections)

    setTimeout(() => {
        console.log('Node connection counts after making connections: \n\n \t\t', pg.Client.connectionMap, '\n')
    }, 2000)

    setTimeout(async() => {
        
        let ans = await askQuestion("Press Enter to proceed to end the connections: ");
        await endNumConnections(numConnections, clientArray);
        setTimeout(() => {
            console.log('Node connection counts after ending the connections: \n\n \t\t', pg.Client.connectionMap, '\n')
        }, 2000)

      setTimeout(async() => {
        ans = await askQuestion("Press Enter after stopping one node and want to proceed to Next iteration to make connections: ");
        clientArray = await createNumConnections(numConnections);
        setTimeout(() => {
            console.log('Node connection counts after some node is down and connections are made: \n\n \t\t', pg.Client.connectionMap, '\n')
            process.exit();
        }, 2000)
      },2000)     
    }, 3000)
    
})();





