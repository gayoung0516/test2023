package chaincode

import (
	"encoding/json"
	"fmt"
	"time"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// 삼성 제품 정보
type Product struct {
	Name    	   string `json:"name"` //제품명 - 냉장고
	Model		   string `json:"model"` //제품모델 - BS01
	Code 		   string `json:"code"` //제품일련번호 제품코드 - BS0101
	Purchase       int    `json:"purchase"` //구입날짜 - 2023.11.22
	Finaldate	   int    `json:"finaldate"`//보증기간 - 2024.11.21
}

// 이용자 정보
type User struct {
	ID             string   `json:"userID"` //회원가입 ID - user1
	List		   []string `json:"list"` //보유한 제품리스트 - 냉장고, 티비, 에어컨
}

type QueryResult struct {
	Key    string `json:"Key"`
	Record *Product
}


// 제품등록 => 제품 이름, 제품 모델, 제품 코드(자동생성), 제품 소유주, 제품 구매기간 및 보증기간 (자동생성)
func (s *SmartContract) AddProduct(ctx contractapi.TransactionContextInterface, name string, model string, userID string) error {
	
	
	//제품 일련번호 자동생성
	assetJSON, err := ctx.GetStub().GetState("codecount")
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		ctx.GetStub().PutState("codecount", []byte("0"))	
	}
	codecountINT,_ := strconv.Atoi(string(assetJSON))
	codecountINT += 1

	codeID := "SM202311"+strconv.Itoa(codecountINT)

	nowTime := time.Now()
	unixTime := int(nowTime.Unix()) //현재시간에 유닉스타임을 넣는다..

	product := Product{
		Name:           name,
		Model:          model,
		Code :			codeID,
		Purchase:       unixTime,
		Finaldate:		unixTime + (31536000), //우선 어떻게 할지 몰라서 57번째랑 똑같이 + 1년
	}
		

	assetJSON, err = json.Marshal(product)
	if err != nil {
		return err
	}

	ctx.GetStub().PutState("codecount",[]byte(strconv.Itoa(codecountINT)))
	ctx.GetStub().PutState(codeID, assetJSON)
	
	if err != nil {
		return err
	}


	//1. User 정보 조회해보기
	assetJSON, err = ctx.GetStub().GetState(userID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	
	user := new(User)

	// 2. User 정보 갱신 : 등록된 상품을 User 정보의 리스트에 올린다.
	// User 정보가 이미 원장에 있을경우
	if assetJSON != nil {
		err = json.Unmarshal(assetJSON, &user)
		if err != nil {
			return err
		}
		user.List = append(user.List,codeID)
	} else { // User정보가 원장에 없을경우
		user.ID = userID
		user.List = append(user.List,codeID)
	}

	// 3. 갱신된 User정보 원장에 저장
	assetJSON, err = json.Marshal(user)
	if err != nil {
		return err
	}	
	ctx.GetStub().PutState(userID, assetJSON)

	return nil
}

//제품 조회 ----> 제품하나짜리 조회함??
func (s *SmartContract) QueryProduct(ctx contractapi.TransactionContextInterface, codeID string) (*Product, error) {
	assetJSON, err := ctx.GetStub().GetState(codeID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", codeID)
	}

	var product Product
	err = json.Unmarshal(assetJSON, &product)
	if err != nil {
		return nil, err
	}

	return &product, nil

}




//제품거래(소유주 변경)
func (s *SmartContract) TransProduct(ctx contractapi.TransactionContextInterface, codeID string, oldUserID string, newUserID string) error {
	// 제품정보의 주인정보를 바꿔서 원장에 저장
	product, err := s.QueryProduct(ctx, codeID)

	if err != nil {
		return err
	}

	assetJSON, _ := json.Marshal(product)

	err = ctx.GetStub().PutState(codeID, assetJSON)
	if err != nil {
		return err
	}


	// User정보(판매자)의 보유 제품리스트에서 해당 codeID 제거
	assetJSON, err = ctx.GetStub().GetState(oldUserID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return fmt.Errorf("the asset %s does not exist", oldUserID)
	}

	var user User
	err = json.Unmarshal(assetJSON, &user)
	if err != nil {
		return err
	}

	var newList []string

	for _, code := range user.List{
	if code != codeID {
		newList = append(newList, code)
		}
	}

	user.List = newList

	// 3. 갱신된 User정보 원장에 저장
	assetJSON, err = json.Marshal(user)
	if err != nil {
		return err
	}	
	err = ctx.GetStub().PutState(oldUserID, assetJSON)
	if err != nil {
		return err
	}


	// 4. User정보(구매자:newOwner)의 보유 제품리스트에서 해당 codeID 추가

	assetJSON, err = ctx.GetStub().GetState(newUserID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return fmt.Errorf("the asset %s does not exist", newUserID)
	}
	err = json.Unmarshal(assetJSON, &user)
	if err != nil {
		return err
	}

	user.List = append(user.List, codeID)

	assetJSON, err = json.Marshal(user)
	if err != nil {
		return err
	}	
	err = ctx.GetStub().PutState(newUserID, assetJSON)
	if err != nil {
		return err
	}

	return nil

	}



// //제품거래(소유주 변경)
// func (s *SmartContract) TransProduct(ctx contractapi.TransactionContextInterface, codeID string, oldOwner string, newOwner string, newOwnerName string) error {
// 	// 제품정보의 주인정보를 바꿔서 원장에 저장
// 	product, err := s.QueryProduct(ctx, codeID)

// 	if err != nil {
// 		return err
// 	}

// 	product.Owner = newOwnerName

// 	assetJSON, _ := json.Marshal(product)

// 	err = ctx.GetStub().PutState(codeID, assetJSON)
// 	if err != nil {
// 		return err
// 	}

	
// 	// User정보(판매자)의 보유 제품리스트에서 해당 codeID 제거
// 	assetJSON, err = ctx.GetStub().GetState(oldOwner)
// 	if err != nil {
// 		return fmt.Errorf("failed to read from world state: %v", err)
// 	}
// 	if assetJSON == nil {
// 		return fmt.Errorf("the asset %s does not exist", oldOwner)
// 	}
// 	var user User
// 	err = json.Unmarshal(assetJSON, &user)
// 	if err != nil {
// 		return err
// 	}

// 	var newList []string

// 	for _, code := range user.List{
// 		if code != codeID {
// 			newList = append(newList, code)
// 		}
// 	}
// 	user.List = newList

// 	// 3. 갱신된 User정보 원장에 저장
// 	assetJSON, err = json.Marshal(user)
// 	if err != nil {
// 		return err
// 	}	
// 	err = ctx.GetStub().PutState(oldOwner, assetJSON)
// 	if err != nil {
// 		return err
// 	}


// 	// 4. User정보(구매자:newOwner)의 보유 제품리스트에서 해당 codeID 추가

// 	assetJSON, err = ctx.GetStub().GetState(newOwner)
// 	if err != nil {
// 		return fmt.Errorf("failed to read from world state: %v", err)
// 	}
// 	if assetJSON == nil {
// 		return fmt.Errorf("the asset %s does not exist", newOwner)
// 	}
// 	err = json.Unmarshal(assetJSON, &user)
// 	if err != nil {
// 		return err
// 	}

// 	user.List = append(user.List, codeID)

// 	assetJSON, err = json.Marshal(user)
// 	if err != nil {
// 		return err
// 	}	
// 	err = ctx.GetStub().PutState(newOwner, assetJSON)
// 	if err != nil {
// 		return err
// 	}

// 	return nil

// }



// 본인 보유제품리스트 조회하기
func (s *SmartContract) QueryOwnedProduct(ctx contractapi.TransactionContextInterface, userID string) ([]string,error) {

	listAsBytes, err := ctx.GetStub().GetState(userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to read from world state. %s", err.Error())
	}
	if listAsBytes == nil {
		return nil, nil
	}

	user := new(User)
	_ = json.Unmarshal(listAsBytes, user)

	return user.List, nil	
	
}


// 관리자 : 모든 제품 리스트 조회하기
func (s *SmartContract) QueryAllProduct(ctx contractapi.TransactionContextInterface) ([]QueryResult, error) {
	startKey := "SM202311"
	endKey := "SM202319999"

	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)

	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	results := []QueryResult{}

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()

		if err != nil {
			return nil, err
		}

		product := new(Product)
		_ = json.Unmarshal(queryResponse.Value, product)

		queryResult := QueryResult{Key: queryResponse.Key, Record: product}
		results = append(results, queryResult)
	}

	return results, nil
}