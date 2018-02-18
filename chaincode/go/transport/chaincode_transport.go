
package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"encoding/pem"
	"crypto/x509"
	"strings"
	"encoding/json"
)

var logger = shim.NewLogger("TransportChaincode")

type Status struct {
	Name string `json:"name"`
	Author string `json:"author"`
}

type Transport struct {
	Id string `json:"id"`
	Qty int `json:"qty"`
	Statuses []Status `json:"statuses"`
}

// TransportChaincode example simple Chaincode implementation
type TransportChaincode struct {
}

func (t *TransportChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debug("Init")

	return shim.Success(nil)
}

func (t *TransportChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debug("Invoke")

	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return shim.Error(err.Error())
	}

	name, org := getCreator(creatorBytes)

	logger.Debug("transaction creator " + name + "@" + org)

	function, args := stub.GetFunctionAndParameters()
	if function == "create" {
		return t.create(stub, args)
	} else if function == "update" {
		return t.update(stub, args, org)
	} else if function == "query" {
		return t.query(stub, args)
	}

	return pb.Response{Status:403, Message:"Invalid invoke function name."}
}

func (t *TransportChaincode) create(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	var transport Transport

	transportString := args[0]

	err := json.Unmarshal([]byte(transportString), &transport)
	if err != nil {
		return shim.Error(err.Error())
	}

	key, err := stub.CreateCompositeKey("Transport", []string{transport.Id})
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(key, []byte(transportString))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (t *TransportChaincode) update(stub shim.ChaincodeStubInterface, args []string, org string) pb.Response {
	if len(args) != 2 {
		return pb.Response{Status:403, Message:"Incorrect number of arguments"}
	}

	id := args[0]
	statusString := args[1]

	key, err := stub.CreateCompositeKey("Transport", []string{id})
	if err != nil {
		return shim.Error(err.Error())
	}

	transportBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	var transport Transport

	err = json.Unmarshal(transportBytes, &transport)
	if err != nil {
		return shim.Error(err.Error())
	}

	status := Status{Name: statusString, Author: org}

	transport.Statuses = append(transport.Statuses, status)

	transportBytes, err = json.Marshal(transport)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(key, transportBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (t *TransportChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	it, err := stub.GetStateByPartialCompositeKey("Transport", []string{})
	if err != nil {
		return shim.Error(err.Error())
	}

	defer it.Close()

	transports := []Transport{}

	for it.HasNext() {
		next, err := it.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		var transport Transport
		err = json.Unmarshal(next.Value, &transport)
		if err != nil {
			return shim.Error(err.Error())
		}

		transports = append(transports, transport)
	}

	result, err := json.Marshal(transports)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(result)
}

var getCreator = func (certificate []byte) (string, string) {
	data := certificate[strings.Index(string(certificate), "-----"): strings.LastIndex(string(certificate), "-----")+5]
	block, _ := pem.Decode([]byte(data))
	cert, _ := x509.ParseCertificate(block.Bytes)
	organization := cert.Issuer.Organization[0]
	commonName := cert.Subject.CommonName
	logger.Debug("commonName: " + commonName + ", organization: " + organization)

	organizationShort := strings.Split(organization, ".")[0]

	return commonName, organizationShort
}

func main() {
	err := shim.Start(new(TransportChaincode))
	if err != nil {
		logger.Error(err.Error())
	}
}
