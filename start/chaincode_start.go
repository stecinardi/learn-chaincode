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
	//"time"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"crypto/x509"
	"encoding/pem"
	"net/url"
	"strconv"
	"strings"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}


//asset
type Watch struct {
	Serial string  	   			`json:"serial"`
	Price string 	   			`json:"price"`
	Model string 	   			`json:"model"`
	Actor string 	   			`json:"actor"`
	Status int					`json:"status"`
	Secret string       		`json:"secret"`
	Authenticated bool  		`json:"authenticated"`
	Attachments []Attachment 	`json:"attachments"`
	Loyalties []Loyalty			`json:"loyalties"`
}

type Loyalty struct {
	Status int 					`json:"status"`
	StartDate string  			`json:"startDate"`
	EndDate string 				`json:"endDate"`
	Description string 			`json:"description"`
	Type string 				`json:"type"`
}

type Attachment struct {
	Id string 		`json:"id"`
	URL string 		`json:"url"`
}

type Role string

const (
	manifacturer 	= 1
	distributor 	= 2
	retailer		= 3
)

type Actor struct {
	Name string `json:"name"`
	Description string `json:"description"`
	Role Role `json:"role"`
}

type User struct {
	CodCliente string `json:"codCliente"`
	Watches []string `json:"watches"` //contiene i seriali degli orologi in suo possesso
}

type User_and_eCert struct {
	Identity string `json:"identity"`
	eCert string `json:"ecert"`
}

type Response struct {
	Status int `json:"status"`
	Code string `json:"code"`
	Message string `json:"message"`
}

var watchIndexStr = "_watchindex"
var userIndexStr = "_userindex"

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resetta tutto
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

//inizializzo la lista di indici dei vari orologi contenuti nella blockchain
	var err error
	var empty []string
	watchIndexJsonAsBytes, _ := json.Marshal(empty)								//marshal an emtpy array of strings to clear the index
	err = stub.PutState(watchIndexStr, watchIndexJsonAsBytes)
	if err != nil {
		return nil, err
	}

	userIndexJsonAsBytes, _ := json.Marshal(empty)								//marshal an emtpy array of strings to clear the index
	err = stub.PutState(userIndexStr, userIndexJsonAsBytes)
	if err != nil {
		return nil, err
	}

	for i:=0; i < len(args); i=i+2 {
		t.add_ecert(stub, args[i], args[i+1])
	}

	return nil,nil
}

//==============================================================================================================================
//	 Router Functions
//==============================================================================================================================
//	Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//		  initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {													//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "move_to_next_actor" {
		return t.moveToNextActor(stub,args)
	} else if function == "create_watch" {
		return t.createWatch(stub,args)
	} else if function == "add_attachment" {
		return t.addAttachment(stub,args)
	} else if function == "register_watch" {
		return t.registerWatch(stub,args)
	} else if function == "authenticate_watch" {
		return t.authenticateWatch(stub,args)
	} else if function == "addLoyalty" {
		return t.addLoyalty(stub,args)
	}
	

	fmt.Println("invoke did not find func: " + function)					//error

	return nil, errors.New("Received unknown function invocation")
}


// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" {											//read a variable
		return t.read(stub,args)
	} else if function == "read_all_watches" {
		return t.readAllWatches(stub,args)
	}else if function == "read_all_users" {
		return t.readAllUsers(stub,args)
	} else if function == "get_caller_data" {
		return t.get_caller_data(stub)
	} else if function == "is_authenticated_watch" {
		return t.isAuthenticatedWatch(stub,args)
	} else if function == "verify_authenticate_watch" {
		return t.verify_authenticateWatch(stub,args)
	} else if function == "verify_register_watch" {
		return t.verify_registerWatch(stub,args)
	} else if function == "loyalties_per_watch" {
		return t.loyalties_per_watch(stub,args)
	}

	fmt.Println("query did not find func: " + function)					

	return nil, errors.New("Received unknown function query")
}

func (t *SimpleChaincode) read (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
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

func (t *SimpleChaincode) loyalties_per_watch (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var serial,jsonResp string
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")

	}
	serial = args[0]
	fmt.Println("serial: " + serial)
	watchAsBytes, err := stub.GetState(serial)

	 if err != nil {
        jsonResp = "{\"Error\":\"Failed to get state for " + serial + "\"}"
        return nil, errors.New(jsonResp)
    }
	
	var watch Watch
	json.Unmarshal(watchAsBytes, &watch)
	var loyalties [] Loyalty = watch.Loyalties

	jsonAsBytes, err := json.Marshal(loyalties)
	if err != nil {
		return nil, err
	}

	return jsonAsBytes,nil

}


func (t *SimpleChaincode) readAllWatches (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	watchIndexAsBytes, err := stub.GetState(watchIndexStr)
		if err != nil {
			return nil, errors.New("Failed to get watch index")
		}

	var watchIndex []string
	json.Unmarshal(watchIndexAsBytes, &watchIndex)

	var allWatches []Watch
	for _, x := range watchIndex {
		var watch Watch
		watchAsBytes, err := stub.GetState(x)
		if err != nil {
			return nil, errors.New("Failed to get watch")
		}
		json.Unmarshal(watchAsBytes, &watch)
        allWatches = append (allWatches,watch)
    }

    jsonAsBytes, err := json.Marshal(allWatches)
	if err != nil {
		return nil, err
	}

	return jsonAsBytes,nil
}

func (t *SimpleChaincode) readAllUsers (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	userAsBytes, err := stub.GetState(userIndexStr)
		if err != nil {
			return nil, errors.New("Failed to get watch index")
		}
		return userAsBytes,nil
}

func (t *SimpleChaincode) isAuthenticatedWatch (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var serial = args[0]
	var secret = args[1]

	watchIndexAsBytes, err := stub.GetState(watchIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get watch index")
	}

	var watchIndex []string
	json.Unmarshal(watchIndexAsBytes, &watchIndex)

	if !stringInSlice(serial, watchIndex) {
		return nil,errors.New ("Watch serial not exists. Verify the serial and please try again")
	}

	//verifichiamo lo stato di autenticazione dell'orologio - è già stato autenticato da un altro utente?
	
	watchAsBytes, err := stub.GetState(serial)
	if err != nil {
		return nil, err
	}
	var response Response

	watch := unmarshWatchJson(watchAsBytes)

	if watch.Authenticated == true && secret == watch.Secret {
		
		response.Status = 0
		response.Message = `{ "uuid": "`+ watch.Actor + `", "authenticated" : "` + strconv.FormatBool(watch.Authenticated) + `"}`
	} else {

		response.Status = -1
		response.Message = `{ "authenticated" : "` + strconv.FormatBool(watch.Authenticated) + `"}`
	
	}

	jsonAsBytes, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return jsonAsBytes, nil
}

func (t *SimpleChaincode) verify_authenticateWatch (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	
	var serial = args[0]
	var secret = args[1]

	var response Response

	watchIndexAsBytes, err := stub.GetState(watchIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get watch index")
	}

	var watchIndex []string
	json.Unmarshal(watchIndexAsBytes, &watchIndex)

	//verifichiamo lo stato di autenticazione dell'orologio - è già stato autenticato da un altro utente?
	
	watchAsBytes, err := stub.GetState(serial)
	
	if err != nil {
		return nil, err
	}

	watch := unmarshWatchJson(watchAsBytes)

	if !stringInSlice(serial, watchIndex) {
		
		response.Status = -1
		response.Code = "00001" // incorrect serial or watch not present
		response.Message = `{ "description" : "incorrect serial or watch not exists" }`
		
	} else if len(watch.Secret) <= 0 ||  watch.Secret != secret {
		
		response.Status = -1
		response.Code = "00002" // no secret or incorrect secret 
		response.Message = `{ "description": "no secret or incorrect secret" }`
	} else if watch.Authenticated == true {

		response.Status = -1
		response.Code = "00003" //watch already authenticated
		response.Message = `{ "description" : "watch already authenticated"}`
	
	} else {
		response.Status = 0
		response.Code = "" //watch already authenticated
		response.Message = `{ "description" : "watch can be authenticated"}`
	}

	jsonAsBytes, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return jsonAsBytes, nil

}


func (t *SimpleChaincode) authenticateWatch (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var serial = args[0]
	var userId = args[1]

	watchAsBytes, err := stub.GetState(serial)
	
	if err != nil {
		return nil, err
	}

	watch := unmarshWatchJson(watchAsBytes)

	watch.Actor = userId
	watch.Authenticated = true

	jsonString, err := json.Marshal(watch)
	if err != nil {
		fmt.Println("error: ", err)
	}

	err = stub.PutState(serial, jsonString)

	if err != nil {
		return nil,err
	}

	return nil, nil
}


func (t *SimpleChaincode) createWatch (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var key = args [0]
	var jsonBlob = []byte(args[1])

	watch := unmarshWatchJson(jsonBlob)
	watch.Authenticated = false
	watch.Status = 0

	fmt.Println("running createWatch() - actor: " + watch.Actor)
	fmt.Printf("watch object: %+v", watch)

	//controlliamo se il seriale è già stato registrato in precedenza

	watchIndexAsBytes, err := stub.GetState(watchIndexStr)
		if err != nil {
			return nil, errors.New("Failed to get watch index")
		}

	var watchIndex []string
	json.Unmarshal(watchIndexAsBytes, &watchIndex)

	if stringInSlice(key, watchIndex) {
		return nil, errors.New("Watch serial already exists. Change serial number and please try again")
	}

	jsonString, err := json.Marshal(watch)
	if err != nil {
		fmt.Println("error: ", err)
	}

	err = stub.PutState(key, jsonString)

	if err != nil {
		return nil,err
	}

	//get index array

	/*watchAsBytes, err := stub.GetState(watchIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get watch index")
	}

	var watchIndex []string
	json.Unmarshal(watchAsBytes, &watchIndex)							//un stringify it aka JSON.parse()
	*/

	//append

	watchIndex = append(watchIndex, key)								//add watch name to index list
	fmt.Println("! watch index: ", watchIndex)
	jsonAsBytes, _ := json.Marshal(watchIndex)
	err = stub.PutState(watchIndexStr, jsonAsBytes)						//store name of watch

	fmt.Println("- end create new watch")

	return nil, nil
}

func (t *SimpleChaincode) verify_registerWatch (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting serial")
	}

	var serial = args[0]
	var response Response

	watchIndexAsBytes, err := stub.GetState(watchIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get watch index")
	}

	var watchIndex []string
	json.Unmarshal(watchIndexAsBytes, &watchIndex)

	//verifichiamo lo stato di autenticazione dell'orologio - è già stato autenticato da un altro utente?
	
	watchAsBytes, err := stub.GetState(serial)
	
	if err != nil {
		return nil, err
	}

	watch := unmarshWatchJson(watchAsBytes)

	if !stringInSlice(serial, watchIndex) {
		response.Status = -1
		response.Code = "00001" // incorrect serial or watch not present
		response.Message = `{ "description": "incorrect serial or watch not exists" }` 
	} else if len(watch.Secret) > 0 {

		response.Status = -1
		response.Code = "00004" // incorrect serial or watch not present
		response.Message = `{ "description": "watch serial already registered for a customer" }` 

	} else if watch.Authenticated == true {

		response.Status = -1
		response.Code = "00003" // incorrect serial or watch not present
		response.Message = `{ "description" : "watch already authenticated" }` 

	} else {
		response.Status = 0
		response.Code = "" //watch already authenticated
		response.Message = `{ "description" : "watch can be registered"}`

	}

	jsonAsBytes, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return jsonAsBytes, nil

}

func (t *SimpleChaincode) registerWatch (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting serial and customer code")
	}

	var serial = args[0]
	var secret = args[1]
	
	watchAsBytes, err := stub.GetState(serial)
	if err != nil {
		return nil, err
	}

	watch := unmarshWatchJson(watchAsBytes)

	//registriamo l'hash secret dell'orologio su blockchain

	watch.Secret = secret
	
	jsonString, err := json.Marshal(watch)
	if err != nil {
		fmt.Println("error: ", err)
	}

	err = stub.PutState(serial, jsonString)

	if err != nil {
		return nil,err
	}

	fmt.Println("- end registerWatch function -")

	return nil, nil

}

func (t *SimpleChaincode) addAttachment (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	fmt.Println("running addAttachment() for the watch with serial: " + args[0])

	if len(args) != 3 {
			return nil, errors.New("Incorrect number of arguments. Expecting serial, attachment id and attachment URL")
	}

	var attachment Attachment
	serialWatch := args[0] // id orologio
	attachment.Id = args[1]
	attachment.URL = args[2]
	watchAsBytes, err := stub.GetState(serialWatch)
	if err != nil {
		return nil, err
	}

	watch := unmarshWatchJson(watchAsBytes)
	watch.Attachments = append (watch.Attachments,attachment)

	jsonAsBytes, err := json.Marshal(watch)
	if err != nil {
		fmt.Println("error: ", err)
	}

	err = stub.PutState(args[0], jsonAsBytes)								//rewrite the watch with id as key
	if err != nil {
		return nil, err
	}

	return nil, nil

}

func (t *SimpleChaincode) addLoyalty (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	fmt.Println("running addLoyalty() for the watch with serial: " + args[0])

	if len(args) != 2 {
			return nil, errors.New("Incorrect number of arguments. Expecting serial, attachment id and attachment URL")
	}

	var loyalty Loyalty
	var serialWatch = args[0] // id orologio
	var jsonBlob = []byte(args[1])

	loyalty = unmarshLoyaltyJson(jsonBlob);

	watchAsBytes, err := stub.GetState(serialWatch)
	if err != nil {
		return nil, err
	}

	watch := unmarshWatchJson(watchAsBytes)
	watch.Loyalties = append (watch.Loyalties,loyalty)

	jsonAsBytes, err := json.Marshal(watch)
	if err != nil {
		fmt.Println("error: ", err)
	}

	err = stub.PutState(serialWatch, jsonAsBytes)								//rewrite the watch with id as key
	
	if err != nil {
		return nil, err
	}

	return nil, nil

}

func (t *SimpleChaincode) moveToNextActor (stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting serial and next actor as arguments")
	}

	idWatch := args[0] // id orologio
	nextActor := args[1]

	fmt.Println("running moveToNextActor() for the watch with serial: " + args[0])

	var watch Watch

	watchAsBytes, err := stub.GetState(idWatch)

	watch = unmarshWatchJson(watchAsBytes)
	watch.Actor = nextActor
	watch.Status = watch.Status + 1
	if err != nil {
		return nil, err
	}
	jsonString, err := json.Marshal(watch)
	if err != nil {
		fmt.Println("error: ", err)
	}

	err = stub.PutState(watch.Serial, jsonString)

	if err != nil {
		return nil,err
	}

	fmt.Println("Watch with serial: " + args[0] + " moved to " + nextActor)

	return nil,nil

}

func  unmarshWatchJson (jsonAsByte []byte) (Watch) {
	var watch Watch
	err := json.Unmarshal(jsonAsByte, &watch)
	if err != nil {
		fmt.Println("error:", err)
	}
	return watch
}

func  unmarshLoyaltyJson (jsonAsByte []byte) (Loyalty) {
	var loyalty Loyalty
	err := json.Unmarshal(jsonAsByte, &loyalty)
	if err != nil {
		fmt.Println("error:", err)
	}
	return loyalty
}

func  unmarshUserJson (jsonAsByte []byte) (User) {
	var user User
	err := json.Unmarshal(jsonAsByte, &user)
	if err != nil {
		fmt.Println("error:", err)
	}
	return user
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}


//==============================================================================================================================
//	 General Functions
//==============================================================================================================================
//	 get_ecert - Takes the name passed and calls out to the REST API for HyperLedger to retrieve the ecert
//				 for that user. Returns the ecert as retrived including html encoding.
//==============================================================================================================================
func (t *SimpleChaincode) get_ecert(stub shim.ChaincodeStubInterface, name string) ([]byte, error) {

	ecert, err := stub.GetState(name)

	if err != nil { return nil, errors.New("Could not retrieve ecert for user " + name) }

	return ecert, nil
}

//==============================================================================================================================
//	 add_ecert - Adds a new ecert and user pair to the table of ecerts
//==============================================================================================================================

func (t *SimpleChaincode) add_ecert(stub shim.ChaincodeStubInterface, name string, ecert string) ([]byte, error) {


	err := stub.PutState(name, []byte(ecert))

	if err == nil {
		return nil, errors.New("Error storing eCert for user " + name + " identity: " + ecert)
	}

	return nil, nil

}

//==============================================================================================================================
//	 get_caller - Retrieves the username of the user who invoked the chaincode.
//				  Returns the username as a string.
//==============================================================================================================================

func (t *SimpleChaincode) get_username(stub shim.ChaincodeStubInterface) (string, error) {

	bytes, err := stub.GetCallerCertificate();
	if err != nil {
		return "", errors.New("Couldn't retrieve caller certificate")
	}

	x509Cert, err := x509.ParseCertificate(bytes);				// Extract Certificate from result of GetCallerCertificate
	if err != nil {
		return "", errors.New("Couldn't parse certificate")
	}

	return x509Cert.Subject.CommonName, nil
}

//==============================================================================================================================
//	 check_affiliation - Takes an ecert as a string, decodes it to remove html encoding then parses it and checks the
// 				  		certificates common name. The affiliation is stored as part of the common name.
//==============================================================================================================================

func (t *SimpleChaincode) check_affiliation(stub shim.ChaincodeStubInterface, cert string) (int, error) {


	decodedCert, err := url.QueryUnescape(cert);    				// make % etc normal //

	if err != nil {
		return -1, errors.New("Could not decode certificate")
	}
	pem, _ := pem.Decode([]byte(decodedCert))           				// Make Plain text   //
	x509Cert, err := x509.ParseCertificate(pem.Bytes);				// Extract Certificate from argument //

	if err != nil {
		return -1, errors.New("Couldn't parse certificate")
	}

	cn := x509Cert.Subject.CommonName
	res := strings.Split(cn,"\\")
	affiliation, _ := strconv.Atoi(res[2])

	return affiliation, nil

}

//=============================================================================================================================
//	 get_caller_data - Calls the get_ecert and check_role functions and returns the ecert and role for the
//					 name passed. To be implemented
//=============================================================================================================================


func (t *SimpleChaincode) get_caller_data(stub shim.ChaincodeStubInterface) ([]byte, error){

	//get caller data function 

	user, err := t.get_username(stub)
	if err != nil {
		return nil, err
	}

	ecert, err := t.get_ecert(stub, user);
	if err != nil {
		return nil, err
	}

	affiliation, err := t.check_affiliation(stub,string(ecert));
	if err != nil {
		return nil, err
	}

	varToReturn := `{ "user": "`+ user + `", "affiliation" : "` + strconv.Itoa(affiliation) + `"}`

	return []byte(varToReturn), nil

}
