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
type Watch struct {
	Serial string  	`json:"serial"`
	Price string 	`json:"price"`
	Model string 	`json:"model"`
	Actor string 	`json:"actor"`
	Attachments []Attachment
}

type Attachment struct {
	Id string
	URL string
}

type Role string  

const (
	manifacturer 	= 1
	distributor 	= 2
	retailer	= 3
)


type Actor struct {
	Name string
	Description string
	Role Role
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
func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
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
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {													//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	//} else if function == "move_to_actor" {
	//	return t.write(stub,args)
	} else if function == "create_watch" {
		return t.create_watch(stub,args)
	} else if function == "add_attachment" {
		return t.addAttachment(stub,args)
	}

	fmt.Println("invoke did not find func: " + function)					//error

	return nil, errors.New("Received unknown function invocation")
}

func (t *SimpleChaincode) create_watch (stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	
	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the key and value to set")
	}

	var jsonBlob = []byte(args[1])

	var key string	
	
	watch := unmarshJson(jsonBlob)

	fmt.Println("running write() - actor: " + watch.Actor)
	fmt.Printf("watch object: %+v", watch)
	
	key = args [0]

	jsonString, err := json.Marshal(watch)
	if err != nil {
		fmt.Println("error: ", err)
	}

	err = stub.PutState(key, jsonString)
	
	if err != nil {
		return nil,err
	}
	
	return nil, nil
}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" {											//read a variable
		return t.read(stub,args)
	}
	fmt.Println("query did not find func: " + function)						//error

	return nil, errors.New("Received unknown function query")
}

func (t *SimpleChaincode) read (stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var key, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")

	}

	key = args[0]
	fmt.Println("key: " + key)	
	valAsbytes, err := stub.GetState(key)

	 if err != nil {
        jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
        return nil, errors.New(jsonResp)
    }

    return valAsbytes, nil
}

func (t *SimpleChaincode) addAttachment (stub *shim.ChaincodeStub, args []string) ([]byte, error) {
		var attachment Attachment
		idWatch := args[0] // id orologio
		attachment.Id = args[1]
		attachment.URL = args[2]
		watchAsBytes, err := stub.GetState(idWatch)
		if err != nil {
			return nil, err
		}
		watch := unmarshJson(watchAsBytes)
		watch.Attachments = append (watch.Attachments,attachment)

		return nil, nil

}

func  unmarshJson (jsonAsByte []byte) (Watch) {
	var watch Watch
	err := json.Unmarshal(jsonAsByte, &watch)
	if err != nil {
		fmt.Println("error:", err)
	}
	return watch
}

