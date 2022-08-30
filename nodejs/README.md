# Follow the steps to run this example - 
### 1. Getting the smart-driver locally installed 
- Clone the repository using:
```
git clone https://github.com/yugabyte/node-postgres.git
```
- Go to the `node-postgres` folder using:
```
cd node-postgres
```
- Install the node dependencies:
```
npm install 
```
### 2. To verify the Smart-driver features of the smart-driver 
- Get this example locally using:
```
git clone https://github.com/yugabyte/driver-examples.git
```
- Go to the nodejs directory and install dependencies:
```
cd driver-examples/nodejs && npm install
```
- Now, before running the example change the path for smart-driver package with the relative path of your local smart-driver clone in the require function in the following line in all the examples - 
```
const pg = require('../../../node-postgres/packages/pg');
```
- Export environment variable as `YB_PATH` with the value of the relative path of your YugabyteDB installation directory.
```
export YB_PATH = '<relative_path_to_the_YB_installation>'
```
- Run the example from the respective example directory for both `load-balance` and `topology-aware` using:
```
node <example_name>.js
```