package domino

import (
	// "fmt"

	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/stretchr/testify/assert"
)

const localDynamoHost = "http://127.0.0.1:4569"

type UserTable struct {
	DynamoTable
	emailField    String
	passwordField String

	registrationDate Numeric
	loginCount       Numeric
	lastLoginDate    Numeric
	visits           NumericSet
	preferences      Map
	name             String
	lastName         String
	locales          StringSet
	degrees          NumericSet

	registrationDateIndex LocalSecondaryIndex
	nameGlobalIndex       GlobalSecondaryIndex
}

type User struct {
	Email       string            `dynamodbav:"email,omitempty"`
	Password    string            `dynamodbav:"password,omitempty"`
	Visits      []int64           `dynamodbav:"visits,numberset,omitempty"`
	Degrees     []float64         `dynamodbav:"degrees,numberset,omitempty"`
	Locales     []string          `dynamodbav:"locales,stringset,omitempty"`
	LoginCount  int               `dynamodbav:"loginCount,omitempty"`
	LoginDate   int64             `dynamodbav:"lastLoginDate,omitempty"`
	RegDate     int64             `dynamodbav:"registrationDate,omitempty"`
	Preferences map[string]string `dynamodbav:"preferences,omitempty"`
}

func NewUserTable() UserTable {
	pk := StringField("email")
	rk := StringField("password")
	firstName := StringField("firstName")
	lastName := StringField("lastName")
	reg := NumericField("registrationDate")

	nameGlobalIndex := GlobalSecondaryIndex{
		Name:           "name-index",
		PartitionKey:   firstName,
		RangeKey:       lastName,
		ProjectionType: ProjectionTypeALL,
	}

	registrationDateIndex := LocalSecondaryIndex{
		Name:         "registrationDate-index",
		PartitionKey: pk,
		SortKey:      reg,
	}

	return UserTable{
		DynamoTable{
			Name:         "users",
			PartitionKey: pk,
			RangeKey:     rk,
			GlobalSecondaryIndexes: []GlobalSecondaryIndex{
				nameGlobalIndex,
			},
			LocalSecondaryIndexes: []LocalSecondaryIndex{
				registrationDateIndex,
			},
		},
		pk,  //email
		rk,  //password
		reg, //registration
		NumericField("loginCount"),
		NumericField("lastLoginDate"),
		NumericSetField("visits"),
		MapField("preferences"),
		firstName,
		lastName,
		StringSetField("locales"),
		NumericSetField("degrees"),
		registrationDateIndex,
		nameGlobalIndex,
	}
}

func NewDB() DynamoDBIFace {
	region := "us-west-2"
	creds := credentials.NewStaticCredentials("123", "123", "")
	config := aws.
		NewConfig().
		WithRegion(region).
		WithCredentials(creds).
		WithEndpoint(localDynamoHost).
		WithHTTPClient(http.DefaultClient)
	sess := session.New(config)

	return dynamodb.New(sess)
}

func TestCreateTable(t *testing.T) {
	ctx := context.Background()
	db := NewDB()
	table := NewUserTable()

	err := table.CreateTable().ExecuteWith(ctx, db)
	assert.NoError(t, err)

	err = table.DeleteTable().ExecuteWith(ctx, db)
	assert.NoError(t, err)

	// Test nil range key
	table.RangeKey = nil
	table.LocalSecondaryIndexes = nil // Illegal to have an lsi, and no range key
	err = table.CreateTable().ExecuteWith(ctx, db)
	assert.NoError(t, err)

	err = table.DeleteTable().ExecuteWith(ctx, db)
	assert.NoError(t, err)

	// Test nil gsi range key
	table.nameGlobalIndex.RangeKey = nil

	err = table.CreateTable().ExecuteWith(ctx, db)
	assert.NoError(t, err)

	err = table.DeleteTable().ExecuteWith(ctx, db)
	assert.NoError(t, err)

}

func TestGetItem(t *testing.T) {

	ctx := context.Background()

	table := NewUserTable()

	db := NewDB()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	item := User{Email: "naveen@email.com", Password: "password"}
	err = table.PutItem(item).ExecuteWith(ctx, db).Result(nil)
	assert.Nil(t, err)

	var r *User = &User{}
	err = table.GetItem(KeyValue{"naveen@email.com", "password"}).
		SetConsistentRead(true).
		ExecuteWith(ctx, db).
		Result(r)

	assert.Nil(t, err)
	assert.Equal(t, &User{Email: "naveen@email.com", Password: "password"}, r)

}
func TestGetItemEmpty(t *testing.T) {

	table := NewUserTable()

	db := NewDB()

	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	out := table.GetItem(KeyValue{"naveen@email.com", "password"}).ExecuteWith(ctx, db)
	assert.Nil(t, out.Error())
	assert.Empty(t, out.Item)
}

func TestBatchPutItem(t *testing.T) {
	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()
	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	items := []interface{}{}
	for i := 0; i < 100; i++ {
		row := User{Email: fmt.Sprintf("%dbob@email.com", i), Password: "password"}
		items = append(items, row)
	}

	q := table.
		BatchWriteItem().
		PutItems(items...).
		DeleteItems(
			KeyValue{"name@email.com", "password"},
		)
	unprocessed := []*User{}
	f := func() interface{} {
		user := User{}
		unprocessed = append(unprocessed, &user)
		return &user
	}
	err = q.ExecuteWith(ctx, db).Results(f)

	assert.Empty(t, unprocessed)
	assert.NoError(t, err)

	keys := []KeyValue{}
	for i := 0; i < 100; i++ {
		key := KeyValue{fmt.Sprintf("%dbob@email.com", i), "password"}
		keys = append(keys, key)
	}

	g := table.
		BatchGetItem(keys...)

	users := []*User{}
	nextItem := func() interface{} {
		user := User{}
		users = append(users, &user)
		return &user
	}
	err = g.ExecuteWith(ctx, db).Results(nextItem)

	assert.NotEmpty(t, users)
}

func TestBatchGetItem(t *testing.T) {
	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()
	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	u := &User{Email: "bob@email.com", Password: "password"}
	items := []interface{}{u}
	kvs := []KeyValue{}
	for i := 0; i < 198; i++ {
		items = append(items, &User{Email: "bob@email.com", Password: "password" + strconv.Itoa(i)})
		kvs = append(kvs, KeyValue{"bob@email.com", "password" + strconv.Itoa(i)})
	}

	ui := []*User{}
	w := table.BatchWriteItem().PutItems(items...)
	f := func() interface{} {
		u := User{}
		ui = append(ui, &u)
		return &u
	}
	err = w.ExecuteWith(ctx, db).Results(f)

	assert.NoError(t, err)
	assert.Empty(t, ui)

	g := table.BatchGetItem(kvs...)

	users := []*User{}
	nextItem := func() interface{} {
		user := User{}
		users = append(users, &user)
		return &user
	}

	err = g.ExecuteWith(ctx, db).Results(nextItem)

	assert.NoError(t, err)
	assert.Equal(t, len(users), 198)

	// TransactGetItems
	users = []*User{}
	tg := table.TransactGetItems(kvs...)
	b, berr := tg.Build()
	assert.NoError(t, berr)
	assert.Equal(t, 20, len(b))

	err = tg.ExecuteWith(ctx, db).Results(nextItem)
	assert.NoError(t, err)
	assert.Equal(t, len(users), 198)

}

func TestUpdateItem(t *testing.T) {
	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	item := User{Email: "name@email.com", Password: "password", Degrees: []float64{1, 2}, Locales: []string{"eu"}, Preferences: map[string]string{"update_email": "test"}}
	q := table.PutItem(item)
	err = q.ExecuteWith(ctx, db).Result(nil)

	assert.NoError(t, err)

	u := table.
		UpdateItem(KeyValue{"name@email.com", "password"}).
		SetUpdateExpression(
			table.loginCount.Increment(1),
			table.lastLoginDate.SetField(time.Now().UnixNano(), false),
			table.registrationDate.SetField(time.Now().UnixNano(), true),
			table.visits.AddInteger(time.Now().UnixNano()),
			table.preferences.Remove("update_email"),
			table.preferences.Set("test", "value"),
			table.locales.AddString("us"),
			table.degrees.DeleteFloat(1),
		)

	err = u.ExecuteWith(ctx, db).Result(nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	assert.Nil(t, err)
	out := table.GetItem(KeyValue{"name@email.com", "password"}).ExecuteWith(ctx, db)
	assert.NotEmpty(t, out.Item)
	item = User{}
	out.Result(&item)
	assert.Equal(t, item.LoginCount, 1)
	assert.NotNil(t, item.LoginDate)
	assert.NotNil(t, item.RegDate)
	assert.Equal(t, 1, len(item.Visits))
	assert.Equal(t, "value", item.Preferences["test"])
	assert.Equal(t, []float64{2}, item.Degrees)
	assert.Subset(t, []string{"eu", "us"}, item.Locales)
	assert.Subset(t, item.Locales, []string{"eu", "us"})

	u = table.
		UpdateItem(KeyValue{"name@email.com", "password"}).
		SetConditionExpression(table.loginCount.Equals(0)).
		SetUpdateExpression(table.loginCount.Increment(2))

	failed := u.ExecuteWith(ctx, db).ConditionalCheckFailed()
	out = table.GetItem(KeyValue{"name@email.com", "password"}).ExecuteWith(ctx, db)

	assert.True(t, failed)

}

func TestRemoveAttribute(t *testing.T) {
	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.Nil(t, err)

	q := table.PutItem(User{Email: "brendanr@email.com", Password: "password", LoginCount: 5})
	err = q.ExecuteWith(ctx, db).Result(nil)
	assert.Nil(t, err)

	// remove
	u := table.
		UpdateItem(KeyValue{"brendanr@email.com", "password"}).
		SetUpdateExpression(
			table.registrationDate.SetField(time.Now().UnixNano(), true),
			table.loginCount.RemoveField(),
		)
	err = u.ExecuteWith(ctx, db).Result(nil)
	assert.Nil(t, err)

	g := table.GetItem(KeyValue{"brendanr@email.com", "password"})

	user := &User{}
	err = g.ExecuteWith(ctx, db).Result(user)
	assert.NoError(t, err)
	assert.NotNil(t, user)

	assert.Equal(t, 0, user.LoginCount)
}

func TestPutItem(t *testing.T) {
	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)
	assert.NoError(t, err)

	item := User{Email: "joe@email.com", Password: "password"}
	q := table.PutItem(item).SetConditionExpression(
		And(
			table.emailField.NotExists(),
			table.passwordField.NotExists(),
		),
	)

	err = q.ExecuteWith(ctx, db).Result(nil)
	assert.NoError(t, err)

	v := table.
		UpdateItem(
			KeyValue{"joe@email.com", "password"},
		).
		SetUpdateExpression(
			table.loginCount.Increment(1),
			table.registrationDate.SetField(time.Now().UnixNano(), false),
		)
	err = v.ExecuteWith(ctx, db).Result(nil)

	assert.NoError(t, err)

	g := table.GetItem(KeyValue{"joe@email.com", "password"})

	out := g.ExecuteWith(ctx, db)

	assert.NotEmpty(t, out.Item)
}

func TestTransactWriteItems(t *testing.T) {
	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	users := []User{}
	items := make(map[string]User)
	updates := make(map[interface{}]KeyValue)
	deletes := make(map[interface{}]KeyValue)
	conditions := make(map[interface{}]KeyValue)

	q := table.TransactWriteItems().WithClientRequestToken("token")

	for i := 0; i < 2; i++ {
		ikey := fmt.Sprintf("joe@email%d.com", i)
		items[ikey] = User{Email: ikey, Password: "password", LoginCount: 1}
		users = append(users, items[ikey])
		updates[ikey] = KeyValue{ikey, "password"}
		dkey := fmt.Sprintf("test%d@email.com", i)
		deletes[dkey] = KeyValue{dkey, "password"}
		conditions[ikey] = KeyValue{ikey, "password"}

		q = q.PutItem(items[ikey], table.registrationDate.Equals(123)).
			UpdateItem(updates[ikey], table.emailField.SetField("nonname@email.com", false), table.emailField.Equals(ikey)).
			DeleteItem(deletes[dkey], table.registrationDate.Equals(123)).
			ConditionCheck(conditions[ikey], table.registrationDate.Equals(123))
	}

	var out *dynamodb.TransactWriteItemsInput
	out, err = q.Build()
	assert.NoError(t, err)

	assert.Equal(t, 2*4, len(out.TransactItems))

	assert.Equal(t, aws.String("token"), out.ClientRequestToken)

	for _, it := range out.TransactItems {

		if it.Put != nil {
			v, _ := dynamodbattribute.MarshalMap(items[*it.Put.Item["email"].S])
			assert.Equal(t, v, it.Put.Item)
			assert.NotNil(t, it.Put.ConditionExpression)
		} else if it.Delete != nil {
			m := make(map[string]*dynamodb.AttributeValue)
			appendKeyAttribute(&m, table.DynamoTable, deletes[*it.Delete.Key["email"].S])
			assert.Equal(t, m, it.Delete.Key)
			assert.NotNil(t, it.Delete.ConditionExpression)
		} else if it.Update != nil {
			m := make(map[string]*dynamodb.AttributeValue)
			appendKeyAttribute(&m, table.DynamoTable, updates[*it.Update.Key["email"].S])
			assert.Equal(t, m, it.Update.Key)
			assert.NotNil(t, it.Update.UpdateExpression)
			assert.NotNil(t, it.Update.ConditionExpression)
		} else if it.ConditionCheck != nil {
			m := make(map[string]*dynamodb.AttributeValue)
			appendKeyAttribute(&m, table.DynamoTable, conditions[*it.ConditionCheck.Key["email"].S])
			assert.Equal(t, m, it.ConditionCheck.Key)
			assert.NotNil(t, it.ConditionCheck.ConditionExpression)
		}
	}

	// Put
	qb := table.TransactWriteItems()
	for _, v := range items {
		qb = qb.PutItem(v)
	}

	_, err = qb.ExecuteWith(ctx, db).Results()
	assert.NoError(t, err)

	var results []User

	err = table.Scan().ExecuteWith(ctx, db).Results(func() interface{} {
		results = append(results, User{})
		return &results[len(results)-1]
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, users, results)

	// Update
	q = table.TransactWriteItems()
	for _, v := range updates {
		q = q.UpdateItem(v, table.name.SetField("nonname", false), table.loginCount.Equals(1))
	}

	_, err = q.ExecuteWith(ctx, db).Results()
	assert.NoError(t, err)

	// Condition Check
	q = table.TransactWriteItems()
	for _, v := range conditions {
		q = q.ConditionCheck(v, table.name.Equals("nonname"))
	}
	_, err = q.ExecuteWith(ctx, db).Results()
	assert.NoError(t, err)

	q = table.TransactWriteItems()
	for _, v := range conditions {
		q = q.ConditionCheck(v, table.name.Equals("nonname2"))
	}
	_, err = q.ExecuteWith(ctx, db).Results()
	assert.Error(t, err)

	// Delete
	q = table.TransactWriteItems()
	for _, v := range updates {
		q = q.DeleteItem(v, table.loginCount.Equals(1))
	}

	_, err = q.ExecuteWith(ctx, db).Results()
	assert.NoError(t, err)

	results = nil
	err = table.Scan().ExecuteWith(ctx, db).Results(func() interface{} {
		results = append(results, User{})
		return &results[len(results)-1]
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))

}

func TestExpressions(t *testing.T) {
	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	expr := Or(
		table.registrationDate.Equals(123),
		table.lastName.Contains("25"),
		Not(table.registrationDate.Equals(345)),
		And(
			table.visits.Size(lte, 25),
			table.name.Size(gte, 25),
		),
		table.registrationDate.Equals("test"),
		table.registrationDate.LessThanOrEq("test"),
		table.registrationDate.Between("0", "1"),
		table.registrationDate.In("0", "1"),
	)

	p := table.passwordField.Equals("password")
	q := table.
		Query(
			table.emailField.Equals("name@email.com"),
			&p,
		).
		SetLimit(100).
		SetScanForward(true).
		SetFilterExpression(expr)

	expectedFilter := "registrationDate = :filter_1 OR contains(lastName,:filter_2) OR (NOT registrationDate = :filter_3) OR (size(visits) <=:filter_4 AND size(firstName) >=:filter_5) OR registrationDate = :filter_6 OR registrationDate <= :filter_7 OR (registrationDate between :filter_8 and :filter_9) OR (registrationDate in (:filter_10,:filter_11))"
	assert.Equal(t, expectedFilter, *q.Build().FilterExpression)

	channel := make(chan *User)
	errChan := q.ExecuteWith(ctx, db).StreamWithChannel(channel)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-channel:
			case err = <-errChan:
				return
			}
		}
	}()

	wg.Wait()

	assert.NoError(t, err)
}

func TestDynamoQuery(t *testing.T) {

	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	me := &User{Email: "name@email.com", Password: "password"}
	items := []interface{}{me}
	for i := 0; i < 1000; i++ {
		e := "name@email.com"
		items = append(items, &User{Email: e, Password: "password" + strconv.Itoa(i)})
	}

	ui := []*User{}
	w := table.BatchWriteItem().PutItems(items...)
	f := func() interface{} {
		u := User{}
		ui = append(ui, &u)
		return &u
	}
	err = w.ExecuteWith(ctx, db).Results(f)

	assert.NoError(t, err)

	assert.Empty(t, ui)

	limit := 100
	p := table.passwordField.BeginsWith("password")
	q := table.
		Query(
			table.emailField.Equals("name@email.com"),
			&p,
		).
		SetLimit(limit).
		SetPageSize(10).
		SetScanForward(true)

	results := []*User{}
	err = q.ExecuteWith(ctx, db).Results(func() interface{} {
		r := &User{}
		results = append(results, r)
		return r
	})

	assert.NoError(t, err)
	assert.Equal(t, limit, len(results))

	var values []DynamoDBValue

	for {
		var v []DynamoDBValue
		var lastKey DynamoDBValue
		v, lastKey, err = q.ExecuteWith(ctx, db).ResultsList()

		if len(v) <= 0 || lastKey == nil {
			break
		}
		values = append(values, v...)
		q = q.WithLastEvaluatedKey(lastKey)
		assert.NoError(t, err)
	}
	assert.Equal(t, 1000, len(values))
}

func TestDynamoStreamQuery(t *testing.T) {

	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	me := &User{Email: "name@email.com", Password: "password"}
	items := []interface{}{me}
	for i := 0; i < 1000; i++ {
		e := "name@email.com"
		items = append(items, &User{Email: e, Password: "password" + strconv.Itoa(i)})
	}

	ui := []*User{}
	w := table.BatchWriteItem().PutItems(items...)
	f := func() interface{} {
		u := User{}
		ui = append(ui, &u)
		return &u
	}
	err = w.ExecuteWith(ctx, db).Results(f)

	assert.NoError(t, err)
	assert.Empty(t, ui)

	set := false
	ff := func(c *dynamodb.ConsumedCapacity) {
		set = true
	}

	limit := 10
	p := table.passwordField.BeginsWith("password")
	q := table.
		Query(
			table.emailField.Equals("name@email.com"),
			&p,
		).SetLimit(limit).WithConsumedCapacityHandler(ff).SetScanForward(true)

	users := []User{}
	channel := make(chan *User)

	_ = q.ExecuteWith(ctx, db).StreamWithChannel(channel)

	for u := range channel {
		users = append(users, *u)
	}
	assert.True(t, set)
	assert.NoError(t, err)
	assert.Equal(t, limit, len(users))
}

func TestDynamoQueryError(t *testing.T) {
	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	me := &User{Email: "name@email.com", Password: "password"}
	items := []interface{}{me}
	for i := 0; i < 1000; i++ {
		e := "name@email.com"
		items = append(items, &User{Email: e, Password: "password" + strconv.Itoa(i)})
	}

	_ = table.BatchWriteItem().PutItems(items...).ExecuteWith(ctx, db).Results(nil)

	channel := make(chan *User)
	errChan := table.
		Query(
			table.registrationDate.Equals("name@email.com"),
			nil,
		).
		SetScanForward(true).
		ExecuteWith(ctx, db).
		StreamWithChannel(channel)

	users := []interface{}{}

SELECT:
	for {
		select {
		case u := <-channel:
			if u != nil {
				users = append(users, u)
			} else {
				break SELECT
			}
		case err = <-errChan:
			break SELECT
		}
	}

	assert.NotNil(t, err)

}

func TestDynamoScan(t *testing.T) {

	table := NewUserTable()
	db := NewDB()
	ctx := context.Background()

	err := table.CreateTable().ExecuteWith(ctx, db)
	defer table.DeleteTable().ExecuteWith(ctx, db)

	assert.NoError(t, err)

	me := &User{Email: "name@email.com", Password: "password"}
	items := []interface{}{me}
	for i := 0; i < 1000; i++ {
		e := "name@email.com"
		items = append(items, &User{Email: e, Password: "password" + strconv.Itoa(i)})
	}

	ui := []*User{}
	w := table.BatchWriteItem().PutItems(items...)
	f := func() interface{} {
		u := User{}
		ui = append(ui, &u)
		return &u
	}
	err = w.ExecuteWith(ctx, db).Results(f)

	assert.NoError(t, err)

	assert.Empty(t, ui)

	limit := 1000
	users := []interface{}{}

	channel := make(chan *User)
	q := table.Scan().SetLimit(limit)
	errChan := q.ExecuteWith(ctx, db).StreamWithChannel(channel)

SELECT:
	for {
		select {
		case u := <-channel:
			if u != nil {
				users = append(users, u)
			} else {
				break SELECT
			}
		case err = <-errChan:
			break SELECT
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, limit, len(users))

	var values []DynamoDBValue

	for {
		var v []DynamoDBValue
		var lastKey DynamoDBValue
		v, lastKey, err = q.ExecuteWith(ctx, db).ResultsList()

		if len(v) <= 0 || lastKey == nil {
			break
		}
		values = append(values, v...)
		q = q.WithLastEvaluatedKey(lastKey)
		assert.NoError(t, err)
		assert.Equal(t, limit, len(v))
	}

	values = append(values, values...)
	assert.True(t, len(values) >= limit)
}
