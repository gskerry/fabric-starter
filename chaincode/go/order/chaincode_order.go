
package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"encoding/pem"
	"crypto/x509"
	"strings"
	"encoding/json"
)

var logger = shim.NewLogger("OrderChaincode")

type Order struct {
	Id string `json:"id"`
	Price float32 `json:"price"`
	Qty int `json:"qty"`
	Status string `json:"status"`
}

// OrderChaincode example simple Chaincode implementation
type OrderChaincode struct {
}

func (t *OrderChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debug("Init")

	return shim.Success(nil)
}

func (t *OrderChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debug("Invoke")

	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return shim.Error(err.Error())
	}

	name, org := getCreator(creatorBytes)

	logger.Debug("transaction creator " + name + "@" + org)

	function, args := stub.GetFunctionAndParameters()
	if function == "create" {
		return t.create(stub, args, org)
	} else if function == "complete" {
		return t.complete(stub, args)
	} else if function == "query" {
		return t.query(stub, args)
	}

	return pb.Response{Status:403, Message:"Invalid invoke function name."}
}

func (t *OrderChaincode) create(stub shim.ChaincodeStubInterface, args []string, org string) pb.Response {

	if org == "retailer" {
		var order Order

		orderString := args[0]

		err := json.Unmarshal([]byte(orderString), &order) //unmarshal it aka JSON.parse()
		if err != nil {
			return shim.Error(err.Error())
		}

		key, err := stub.CreateCompositeKey("Order", []string{order.Id})
		if err != nil {
			return shim.Error(err.Error())
		}

		err = stub.PutState(key, []byte(orderString))
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	} else if org == "distributor" {
		// query retailer-distributor for ready orders
		args := [][]byte{[]byte("query"), []byte("ready")}

		response := stub.InvokeChaincode("order", args, "retailer-distributor")

		logger.Debug(string(response.Payload))

		if response.Status != 200 {
			return shim.Error("Got unexpected return from InvokeChaincode")
		}

		var orders []Order

		err := json.Unmarshal(response.Payload, &orders)
		if err != nil {
			return shim.Error(err.Error())
		}

		for _, order := range orders {
			order.Status = "open"

			productionOrderBytes, err := json.Marshal(order)
			if err != nil {
				return shim.Error(err.Error())
			}

			key, err := stub.CreateCompositeKey("Order", []string{order.Id})
			if err != nil {
				return shim.Error(err.Error())
			}

			err = stub.PutState(key, productionOrderBytes)
			if err != nil {
				return shim.Error(err.Error())
			}
		}
		return shim.Success(nil)

	} else {
		return pb.Response{Status:403, Message:"Don't know how to handle org " + org}
	}
}

func (t *OrderChaincode) complete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return pb.Response{Status:403, Message:"Incorrect number of arguments"}
	}

	id := args[0]

	key, err := stub.CreateCompositeKey("Order", []string{id})
	if err != nil {
		return shim.Error(err.Error())
	}

	orderBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	var order Order

	err = json.Unmarshal(orderBytes, &order)
	if err != nil {
		return shim.Error(err.Error())
	}

	order.Status = "ready"

	orderBytes, err = json.Marshal(order)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(key, orderBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (t *OrderChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) == 1 && args[0] != "ready" && args[0] != "open" {
		// query by id
		id := args[0]

		key, err := stub.CreateCompositeKey("Order", []string{id})
		if err != nil {
			return shim.Error(err.Error())
		}

		orderBytes, err := stub.GetState(key)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(orderBytes)

	} else if len(args) == 1 {
		// query by status
		status := args[0]

		it, err := stub.GetStateByPartialCompositeKey("Order", []string{})
		if err != nil {
			return shim.Error(err.Error())
		}

		defer it.Close()

		orders := []Order{}

		for it.HasNext() {
			next, err := it.Next()
			if err != nil {
				return shim.Error(err.Error())
			}

			var order Order
			err = json.Unmarshal(next.Value, &order)
			if err != nil {
				return shim.Error(err.Error())
			}

			if order.Status == status {
				orders = append(orders, order)
			}
		}

		result, err := json.Marshal(orders)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(result)
	} else {
		return pb.Response{Status:403, Message:"Incorrect number of arguments, provide id or status (open, ready)"}
	}
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
	err := shim.Start(new(OrderChaincode))
	if err != nil {
		logger.Error(err.Error())
	}
}
