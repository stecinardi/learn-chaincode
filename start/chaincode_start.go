/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

//asset
type watch struct {
	id int
	price float64
	color string
	actor string
}

var watchIndexStr = "_watchindex"

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	err:= stub.PutState("hello_world",[]byte(args[0]))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {													//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "write" {
		return t.write(stub,args)
	} else if function == "init_watch" {
		return t.init_watch(stub,args)
	}

	fmt.Println("invoke did not find func: " + function)					//error

	return nil, errors.New("Received unknown function invocation")
}

func (t *SimpleChaincode) write (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key,value string
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the key and value to set")
	}

	key = args [0]
	value = args [1]

	err = stub.PutState(key, []byte (value))
	if err != nil {
		return nil,err
	}
	
	return nil, nil
}

func (t *SimpleChaincode) init_watch (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	
	var err error
	
	/* 	
	 *	EXPECTED PARAMETERS
	 * 	id int
	 *	price float64
	 *	color string
	 *	actor string
	*/

	if len(args) != 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	fmt.Println("running init_watch()")

	i, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("1rd argument must be a numeric string")
	}

	f, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return nil, errors.New("1rd argument must be a numeric string")
	}

	if len(args[2]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}

	if len(args[3]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	
	color := strings.ToLower(args[2])
	actor := strings.ToLower(args[3])

	str := '{"id": "' + strconv.Itoa(args[0]) + '", "color": "' + color + '", "price": ' + strconv.FormatFloat(args[1], 'E', -1, 64) + ', "actor": "' + actor + '"}'
	err = stub.PutState(args[0], []byte(str))								//store marble with id as key
	if err != nil {
		return nil, err
	}

	//get the marble index
	watchAsBytes, err := stub.GetState(watchIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get watch index")
	}
	var watchIndex []string
	json.Unmarshal(watchAsBytes, &watchIndex)							//un stringify it aka JSON.parse()
	
	//append
	watchIndex = append(watchIndex, args[0])								//add marble name to index list
	fmt.Println("! watch index: ", watchIndex)
	jsonAsBytes, _ := json.Marshal(watchIndex)
	err = stub.PutState(watchIndexStr, jsonAsBytes)						//store name of marble

	fmt.Println("- end init watch")
	return nil, nil

	return nil,nil

}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" {											//read a variable
		return t.read(stub,args)
	}
	fmt.Println("query did not find func: " + function)						//error

	return nil, errors.New("Received unknown function query")
}

func (t *SimpleChaincode) read (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")

	}

	key = args[0]
	valAsbytes, err := stub.GetState(key)

	 if err != nil {
        jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
        return nil, errors.New(jsonResp)
    }

    return valAsbytes, nil
}

