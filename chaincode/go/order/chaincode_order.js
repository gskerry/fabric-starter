

const shim = require('fabric-shim');
var logger = shim.NewLogger("OrderChaincode")

let stub = shim.ChaincodeStubInterface

function Init(stub){
	logger.Debug("Init")
	return shim.Success(null)
}

function Invoke(){
    logger.Debug("Invoke")
}

function main() {
	shim.Start(new(OrderChaincode)).then(function(err){
        if err != null {
    		logger.Error(err.Error())
    	}
    })

}
