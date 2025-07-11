package customermodel

import (
	customer "go-shopping-poc/domain/customer/generated"
	"go-shopping-poc/pkg/apiutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// DB to API conversion functions

func ConvertCustomerDBToAPI(dbCustomer customer.CustomersCustomer, dbAddresses []customer.CustomersAddress, dbCreditCards []customer.CustomersCreditcard, dbStatuses []customer.CustomersCustomerstatus) Customer {
	return Customer{
		CustomerID:  dbCustomer.Customerid.String(),
		Username:    dbCustomer.Username,
		Email:       dbCustomer.Email.String,
		FirstName:   dbCustomer.Firstname.String,
		LastName:    dbCustomer.Lastname.String,
		Phone:       dbCustomer.Phone.String,
		Addresses:   ConvertAddressesDBToAPI(dbAddresses),
		CreditCards: ConvertCreditCardsDBToAPI(dbCreditCards),
		Statuses:    ConvertStatusesDBToAPI(dbStatuses),
	}
}

func ConvertAddressesDBToAPI(dbAddresses []customer.CustomersAddress) []Address {
	addresses := make([]Address, len(dbAddresses))
	for i, dbAddr := range dbAddresses {
		addresses[i] = Address{
			AddressType: dbAddr.Addresstype,
			LastName:    dbAddr.Lastname.String,
			Address1:    dbAddr.Address1.String,
			Address2:    dbAddr.Address2.String,
			City:        dbAddr.City.String,
			State:       dbAddr.State.String,
			Zip:         dbAddr.Zip.String,
			IsDefault:   dbAddr.Isdefault.Bool,
		}
	}
	return addresses
}

func ConvertCreditCardsDBToAPI(dbCreditCards []customer.CustomersCreditcard) []CreditCard {
	creditCards := make([]CreditCard, len(dbCreditCards))
	for i, dbCard := range dbCreditCards {
		creditCards[i] = CreditCard{
			CardType:       dbCard.Cardtype.String,
			CardNumber:     dbCard.Cardnumber.String,
			CardHolderName: dbCard.Cardholdername.String,
			CardExpires:    dbCard.Cardexpires.String,
			CardCVV:        dbCard.Cardcvv.String,
			IsDefault:      dbCard.Isdefault.Bool,
		}
	}
	return creditCards
}

func ConvertStatusesDBToAPI(dbStatuses []customer.CustomersCustomerstatus) []CustomerStatus {
	statuses := make([]CustomerStatus, len(dbStatuses))
	for i, dbStatus := range dbStatuses {
		statuses[i] = CustomerStatus{
			Status:         dbStatus.Customerstatus,
			StatusDateTime: dbStatus.Statusdatetime.Time.Format(apiutil.RFC3339Format),
		}
	}
	return statuses
}

// API to DB conversion functions

func ConvertCustomerAPIToDB(apiCustomer Customer) customer.CustomersCustomer {
	return customer.CustomersCustomer{
		Customerid: pgtype.UUID{Bytes: uuid.MustParse(apiCustomer.CustomerID), Valid: true},
		Username:   apiCustomer.Username,
		Email:      apiutil.NullString(apiCustomer.Email),
		Firstname:  apiutil.NullString(apiCustomer.FirstName),
		Lastname:   apiutil.NullString(apiCustomer.LastName),
		Phone:      apiutil.NullString(apiCustomer.Phone),
	}
}

func ConvertAddressesAPIToDB(apiAddresses []Address) []customer.CustomersAddress {
	addresses := make([]customer.CustomersAddress, len(apiAddresses))
	for i, apiAddr := range apiAddresses {
		addresses[i] = customer.CustomersAddress{
			Addresstype: apiAddr.AddressType,
			Firstname:   apiutil.NullString(apiAddr.FirstName),
			Lastname:    apiutil.NullString(apiAddr.LastName),
			Address1:    apiutil.NullString(apiAddr.Address1),
			Address2:    apiutil.NullString(apiAddr.Address2),
			City:        apiutil.NullString(apiAddr.City),
			State:       apiutil.NullString(apiAddr.State),
			Zip:         apiutil.NullString(apiAddr.Zip),
			Isdefault:   pgtype.Bool{Bool: apiAddr.IsDefault, Valid: true},
		}
	}
	return addresses
}

func ConvertCreditCardsAPIToDB(apiCreditCards []CreditCard) []customer.CustomersCreditcard {
	creditCards := make([]customer.CustomersCreditcard, len(apiCreditCards))
	for i, apiCard := range apiCreditCards {
		creditCards[i] = customer.CustomersCreditcard{
			Cardtype:       apiutil.NullString(apiCard.CardType),
			Cardnumber:     apiutil.NullString(apiCard.CardNumber),
			Cardholdername: apiutil.NullString(apiCard.CardHolderName),
			Cardexpires:    apiutil.NullString(apiCard.CardExpires),
			Cardcvv:        apiutil.NullString(apiCard.CardCVV),
			Isdefault:      pgtype.Bool{Bool: apiCard.IsDefault, Valid: true},
		}
	}
	return creditCards
}

func ConvertStatusesAPIToDB(apiStatuses []CustomerStatus) []customer.CustomersCustomerstatus {
	statuses := make([]customer.CustomersCustomerstatus, len(apiStatuses))
	for i, apiStatus := range apiStatuses {
		statuses[i] = customer.CustomersCustomerstatus{
			Customerstatus: apiStatus.Status,
			Statusdatetime: apiutil.NullTimestampFromString(apiStatus.StatusDateTime),
		}
	}
	return statuses
}
