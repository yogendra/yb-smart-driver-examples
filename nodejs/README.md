# Follow the steps to run this example - 
### 1. Install the smart-driver package
```
npm install @yugabytedb/pg
```
### 1. Install the smart-driver Pool package
```
npm install @yugabytedb/pg-pool
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
- Export environment variable as `YB_PATH` with the value of the relative path of your YugabyteDB installation directory.
```
export YB_PATH = '<relative_path_to_the_YB_installation>'
```
- Run the example from the respective example directory for both `load-balance` and `topology-aware` using:
```
node <example_name>.js
```