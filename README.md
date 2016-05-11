# domino
Typesafe DynamoDB query DSL for Go


This is an easy to use wrapper DSL for the aws dynamodb GO api.


```

config := s3.GetAwsConfig("123", "123").WithEndpoint("http://127.0.0.1:8080")
sess := session.New(config)
dynamo := dynamodb.New(sess)

//Define your table schema statically
type MyTable struct {
	DynamoTable
	emailField    dynamoFieldString
	passwordField dynamoFieldString

	registrationDate dynamoFieldNumeric
	loginCount       dynamoFieldNumeric
	lastLoginDate    dynamoFieldNumeric
	vists            dynamoFieldNumericSet
	preferences      dynamoFieldMap
	nameField        dynamoFieldString
	lastNameField    dynamoFieldString

	registrationDateIndex LocalSecondaryIndex
	nameGlobalIndex       GlobalSecondaryIndex
}

type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewMyTable() MyTable {
	pk := DynamoFieldString("email")
	rk := DynamoFieldString("password")
	firstName := DynamoFieldString("firstName")
	lastName := DynamoFieldString("lastName")
	reg := DynamoFieldNumeric("registrationDate")
	return MyTable{
		DynamoTable{
			Name:         "mytable",
			PartitionKey: pk,
			RangeKey:     rk,
		},
		pk,  //email
		rk,  //password
		reg, //registration
		DynamoFieldNumeric("loginCount"),
		DynamoFieldNumeric("lastLoginDate"),
		DynamoFieldNumericSet("visits"),
		DynamoFieldMap("preferences"),
		firstName,
		lastName,
		LocalSecondaryIndex{"registrationDate-index", reg},
		GlobalSecondaryIndex{"name-index", firstName, lastName},
	}
}


table := NewMyTable() 

p := table.PutItem(User{"naveen@email.com","password"}).SetConditionExpression(table.PartitionKey.NotExists()).Build()
r, err := dynamo.PutItem(q)

...

q := table.GetItem(KeyValue{"naveen@email.com", "password"}).SetConsistentRead(true).Build()  //This is type GetItemInput
r, err := dynamo.GetItem(q)

```
