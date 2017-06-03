package domino

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

/*DynamoDBIFace is the interface to the underlying aws dynamo db api*/
type DynamoDBIFace interface {
	CreateTableWithContext(aws.Context, *dynamodb.CreateTableInput, ...request.Option) (*dynamodb.CreateTableOutput, error)
	DeleteTableWithContext(aws.Context, *dynamodb.DeleteTableInput, ...request.Option) (*dynamodb.DeleteTableOutput, error)
	GetItemWithContext(aws.Context, *dynamodb.GetItemInput, ...request.Option) (*dynamodb.GetItemOutput, error)
	BatchGetItemWithContext(aws.Context, *dynamodb.BatchGetItemInput, ...request.Option) (*dynamodb.BatchGetItemOutput, error)
	PutItemWithContext(aws.Context, *dynamodb.PutItemInput, ...request.Option) (*dynamodb.PutItemOutput, error)
	QueryWithContext(aws.Context, *dynamodb.QueryInput, ...request.Option) (*dynamodb.QueryOutput, error)
	ScanWithContext(aws.Context, *dynamodb.ScanInput, ...request.Option) (*dynamodb.ScanOutput, error)
	UpdateItemWithContext(aws.Context, *dynamodb.UpdateItemInput, ...request.Option) (*dynamodb.UpdateItemOutput, error)
	DeleteItemWithContext(aws.Context, *dynamodb.DeleteItemInput, ...request.Option) (*dynamodb.DeleteItemOutput, error)
	BatchWriteItemWithContext(aws.Context, *dynamodb.BatchWriteItemInput, ...request.Option) (*dynamodb.BatchWriteItemOutput, error)
}

// Loader is the interface that specifies the ability to deserialize and load data from dynamodb attrbiute value map
type Loader interface {
	LoadDynamoDBValue(av map[string]*dynamodb.AttributeValue) (err error)
}

func deserializeTo(av map[string]*dynamodb.AttributeValue, item interface{}) (err error) {
	if len(av) <= 0 {
		return
	}

	switch t := (item).(type) {
	case Loader:
		err = t.LoadDynamoDBValue(av)
	default:
		err = dynamodbattribute.UnmarshalMap(av, item)
	}
	return
}

// ToValue is the interface that specifies the ability to serialize data to a value that can be persisted in dynamodb
type ToValue interface {
	ToDynamoDBValue() (bm interface{})
}

func serialize(item interface{}) (err error, av interface{}) {
	switch t := item.(type) {
	case ToValue:
		av = t.ToDynamoDBValue()
	default:
		av, err = dynamodbattribute.MarshalMap(item)
	}
	return
}

const (
	dS    = "S"
	dSS   = "SS"
	dN    = "N"
	dNS   = "NS"
	dB    = "B"
	dBS   = "BS"
	dBOOL = "Bool"
	dNULL = "Null"
	dL    = "List"
	dM    = "M"
)

const (
	ProjectionTypeALL       = "ALL"
	ProjectionTypeINCLUDE   = "INCLUDE"
	ProjectionTypeKEYS_ONLY = "KEYS_ONLY"
)

/*DynamoTable is a static table definition representing a dynamo table*/
type DynamoTable struct {
	Name                   string
	PartitionKey           DynamoFieldIFace
	RangeKey               DynamoFieldIFace //Optional param. If no range key set to EmptyDynamoField()
	GlobalSecondaryIndexes []GlobalSecondaryIndex
	LocalSecondaryIndexes  []LocalSecondaryIndex
}

type DynamoFieldIFace interface {
	Name() string
	Type() string
	IsEmpty() bool
}

type DynamoField struct {
	name  string
	_type string
	empty bool //If true, this represents an empty field
}

type dynamoValueField struct {
	DynamoField
}

type dynamoCollectionField struct {
	DynamoField
}

func (d DynamoField) Name() string {
	return d.name
}
func (d DynamoField) Type() string {
	return d._type
}
func (d DynamoField) IsEmpty() bool {
	return d.empty
}

/*Empty - An empty dynamo field*/
type Empty struct {
	DynamoField
}

/*Numeric - A numeric dynamo field*/
type Numeric struct {
	dynamoValueField
}

/*NumericSet - A numeric set dynamo field*/
type NumericSet struct {
	dynamoCollectionField
}

/*String - A string dynamo field*/
type String struct {
	dynamoValueField
}

/*StringSet - A string set dynamo field*/
type StringSet struct {
	dynamoCollectionField
}

/*Binary - A binary dynamo field*/
type Binary struct {
	dynamoValueField
}

/*BinarySet - A binary dynamo field*/
type BinarySet struct {
	dynamoCollectionField
}

/*Bool - A boolean dynamo field*/
type Bool struct {
	dynamoValueField
}

/*List - A list dynamo field*/
type List struct {
	dynamoCollectionField
}

/*Map - A map dynamo field*/
type Map struct {
	dynamoCollectionField
}

/*EmptyField ... A constructor for an empty dynamo field*/
func EmptyField() Empty {
	return Empty{
		DynamoField{
			empty: true,
			_type: dNULL,
		},
	}
}

/*NumericField ... A constructor for a numeric dynamo field*/
func NumericField(name string) Numeric {
	return Numeric{
		dynamoValueField{
			DynamoField{
				name:  name,
				_type: dN,
			},
		},
	}
}

/*NumericSetField ... A constructor for a numeric set dynamo field*/
func NumericSetField(name string) NumericSet {
	return NumericSet{
		dynamoCollectionField{
			DynamoField{
				name:  name,
				_type: dNS,
			},
		},
	}
}

/*StringField ... A constructor for a string dynamo field*/
func StringField(name string) String {
	return String{
		dynamoValueField{
			DynamoField{
				name:  name,
				_type: dS,
			},
		},
	}
}

/*StringField ... A constructor for a string dynamo field*/
func BoolField(name string) Bool {
	return Bool{
		dynamoValueField{
			DynamoField{
				name:  name,
				_type: dBOOL,
			},
		},
	}
}

/*BinaryField ... A constructor for a binary dynamo field*/
func BinaryField(name string) Binary {
	return Binary{
		dynamoValueField{
			DynamoField{
				name:  name,
				_type: dB,
			},
		},
	}
}

/*BinarySetField ... A constructor for a binary set dynamo field*/
func BinarySetField(name string) BinarySet {
	return BinarySet{
		dynamoCollectionField{
			DynamoField{
				name:  name,
				_type: dBS,
			},
		},
	}
}

/*StringSetField ... A constructor for a string set dynamo field*/
func StringSetField(name string) StringSet {
	return StringSet{
		dynamoCollectionField{
			DynamoField{
				name:  name,
				_type: dSS,
			},
		},
	}
}

/*ListField ... A constructor for a list dynamo field*/
func ListField(name string) List {
	return List{
		dynamoCollectionField{
			DynamoField{
				name:  name,
				_type: dL,
			},
		},
	}
}

/*MapField ... A constructor for a map dynamo field*/
func MapField(name string) Map {
	return Map{
		dynamoCollectionField{
			DynamoField{
				name:  name,
				_type: dL,
			},
		},
	}
}

/*LocalSecondaryIndex ... Represents a dynamo local secondary index*/
type LocalSecondaryIndex struct {
	Name             string
	PartitionKey     DynamoFieldIFace
	SortKey          DynamoFieldIFace
	ProjectionType   string
	NonKeyAttributes []DynamoFieldIFace
}

/*GlobalSecondaryIndex ... Represents a dynamo global secondary index*/
type GlobalSecondaryIndex struct {
	Name             string
	PartitionKey     DynamoFieldIFace
	RangeKey         DynamoFieldIFace //Optional param. If no range key set to EmptyField
	ProjectionType   string
	NonKeyAttributes []DynamoFieldIFace
	ReadUnits        int64
	WriteUnits       int64
}

/*KeyValue ... A Key Value struct for use in GetItem and BatchWriteItem queries*/
type KeyValue struct {
	PartitionKey interface{}
	RangeKey     interface{}
}

/***************************************************************************************/
/************************************** GetItem ****************************************/
/***************************************************************************************/
type get dynamodb.GetItemInput
type output struct {
	*dynamodb.GetItemOutput
	Error error
}

func (o *output) Result(item interface{}) (err error) {
	if o.GetItemOutput == nil {
		return o.Error
	}
	return deserializeTo(o.Item, item)
}

/*GetItem Primary constructor for creating a  get item query*/
func (table DynamoTable) GetItem(key KeyValue) *get {
	q := get(dynamodb.GetItemInput{})
	q.TableName = &table.Name
	appendAttribute(&q.Key, table.PartitionKey.Name(), key.PartitionKey)
	if !table.RangeKey.IsEmpty() {
		appendAttribute(&q.Key, table.RangeKey.Name(), key.RangeKey)
	}
	return &q
}

/*SetConsistentRead ... */
func (d *get) SetConsistentRead(c bool) *get {
	(*d).ConsistentRead = &c
	return d
}

func (d *get) Build() *dynamodb.GetItemInput {
	r := dynamodb.GetItemInput(*d)
	r.ReturnConsumedCapacity = aws.String("INDEXES")
	return &r
}

/**
 ** ExecuteWith ... Execute a dynamo getitem call with a passed in dynamodb instance
 ** dynamo - The underlying dynamodb api
 ** item - The item pointer to be hyderated. I.e. if the table holds User object, item should be a pointer to a uninitialized
 **        User{} struct
 **
 ** Returns a tuple of the hydrated item struct, or an error
 */
func (d *get) ExecuteWith(ctx context.Context, dynamo DynamoDBIFace, opts ...request.Option) (out *output) {

	o, err := dynamo.GetItemWithContext(ctx, d.Build(), opts...)
	if err != nil {
		err = handleAwsErr(err)
		return
	}
	out = &output{
		o,
		nil,
	}

	return
}

type TableName string
type Keys *dynamodb.KeysAndAttributes

/***************************************************************************************/
/************************************** BatchGetItem ***********************************/
/***************************************************************************************/
type batchGet struct {
	input *[]*dynamodb.BatchGetItemInput
	/*A set of mutational operations that might error out, i.e. not pure, and therefore not conducive to a fluent dsl*/
	delayedFunctions []func() error
}

/*BatchGetItem represents dynamo batch get item call*/
func (table DynamoTable) BatchGetItem(items ...KeyValue) *batchGet {
	/*Delay the attribute value construction, until Build time*/
	input := &[]*dynamodb.BatchGetItemInput{}
	delayed := func() error {

		k := make(map[string]*dynamodb.KeysAndAttributes)
		keysAndAttribs := &dynamodb.KeysAndAttributes{}
		k[table.Name] = keysAndAttribs
		ss := []map[string]*dynamodb.KeysAndAttributes{k}

		for i, kv := range items {

			if (i-1)%100 == 99 {
				k = make(map[string]*dynamodb.KeysAndAttributes)
				ss = append(ss, k)

				keysAndAttribs = &dynamodb.KeysAndAttributes{}
				k[table.Name] = keysAndAttribs
			}

			m := map[string]interface{}{
				table.PartitionKey.Name(): kv.PartitionKey,
			}
			if !table.RangeKey.IsEmpty() {
				m[table.RangeKey.Name()] = kv.RangeKey
			}

			attributes, err := dynamodbattribute.MarshalMap(m)

			if err != nil {
				return err
			}

			(*keysAndAttribs).Keys = append((*keysAndAttribs).Keys, attributes)

		}

		for _, m := range ss {
			(*input) = append(*input, &dynamodb.BatchGetItemInput{RequestItems: m})
		}

		return nil
	}

	q := batchGet{
		input:            input,
		delayedFunctions: []func() error{delayed},
	}

	return &q
}

func (d *batchGet) Build() (input []*dynamodb.BatchGetItemInput, err error) {
	for _, function := range d.delayedFunctions {
		err = function()
		if err != nil {
			return
		}
	}
	input = *(d.input)
	for _, i := range input {
		i.ReturnConsumedCapacity = aws.String("INDEXES")
	}

	return
}

/**
 ** ExecuteWith ... Execute a dynamo BatchGetItem call with a passed in dynamodb instance and next item pointer
 ** dynamo - The underlying dynamodb api
 ** nextItem - The item pointer function, which is called on each new object returned from dynamodb. The function should
 ** 		   store each item in an array before returning.
 **
 */
func (d *batchGet) ExecuteWith(ctx context.Context, dynamo DynamoDBIFace, nextItem func() interface{}, opts ...request.Option) error {

	input, err := d.Build()
	if err != nil {
		return err
	}
	for _, bg := range input {
		retry := 0
	Execute:

		out, err := dynamo.BatchGetItemWithContext(ctx, bg, opts...)

		if err != nil {
			return handleAwsErr(err)
		}
		for _, r := range out.Responses {
			for _, item := range r {
				err = dynamodbattribute.UnmarshalMap(item, nextItem())
				if err != nil {
					return handleAwsErr(err)
				}
			}
		}
		if out.UnprocessedKeys != nil && len(out.UnprocessedKeys) > 0 {
			bg.RequestItems = out.UnprocessedKeys
			retry++
			goto Execute
		}
	}

	return nil
}

/***************************************************************************************/
/************************************** PutItem ****************************************/
/***************************************************************************************/
type put dynamodb.PutItemInput

/*PutItem represents dynamo put item call*/
func (table DynamoTable) PutItem(i interface{}) *put {

	q := put(dynamodb.PutItemInput{})
	q.TableName = &table.Name
	q.Item, _ = dynamodbattribute.MarshalMap(i)
	return &q
}

func (d *put) ReturnAllOld() {
	(*dynamodb.PutItemInput)(d).SetReturnValues("ALL_OLD")
}
func (d *put) ReturnNone() {
	(*dynamodb.PutItemInput)(d).SetReturnValues("NONE")
}
func (d *put) SetConditionExpression(c Expression) *put {
	s, m, _ := c.construct(1, true)

	d.ConditionExpression = &s
	d.ExpressionAttributeValues, _ = dynamodbattribute.MarshalMap(m)

	return d
}

func (d *put) Build() *dynamodb.PutItemInput {
	r := dynamodb.PutItemInput(*d)
	return &r
}

/**
 ** ExecuteWith ... Execute a dynamo PutItem call with a passed in dynamodb instance
 ** ctx - An instance of context
 ** dynamo - The underlying dynamodb api
 **
 */
func (d *put) ExecuteWith(ctx context.Context, dynamo DynamoDBIFace, opts ...request.Option) error {
	_, err := dynamo.PutItemWithContext(ctx, d.Build(), opts...)
	if err != nil {
		return handleAwsErr(err)
	}
	return nil
}

/***************************************************************************************/
/************************************** BatchPutItem *********************************/
/***************************************************************************************/
type batchPut struct {
	batches          []dynamodb.BatchWriteItemInput
	table            DynamoTable
	delayedFunctions []func() error
}

/*BatchWriteItem represents dynamo batch write item call*/
func (table DynamoTable) BatchWriteItem() *batchPut {
	r := batchPut{
		batches: []dynamodb.BatchWriteItemInput{},
		table:   table,
	}
	return &r
}

func (d *batchPut) writeItems(putOnly bool, items ...interface{}) *batchPut {
	delayed := func() error {
		batches := []dynamodb.BatchWriteItemInput{}
		batchCount := len(items)/25 + 1
		for i := 1; i <= batchCount; i++ {
			batch := dynamodb.BatchWriteItemInput{
				RequestItems: make(map[string][]*dynamodb.WriteRequest),
			}
			puts := []*dynamodb.WriteRequest{}

			for len(items) > 0 && len(puts) < 25 {
				item := items[0]
				items = items[1:]
				dynamoItem, err := dynamodbattribute.MarshalMap(item)
				if err != nil {
					return err
				}
				var write *dynamodb.WriteRequest
				if putOnly {
					write = &dynamodb.WriteRequest{
						PutRequest: &dynamodb.PutRequest{
							Item: dynamoItem,
						},
					}
				} else {
					write = &dynamodb.WriteRequest{
						DeleteRequest: &dynamodb.DeleteRequest{
							Key: dynamoItem,
						},
					}
				}

				puts = append(puts, write)
			}

			batch.RequestItems[d.table.Name] = puts
			batches = append(batches, batch)

		}
		d.batches = append(d.batches, batches...)
		return nil
	}
	d.delayedFunctions = append(d.delayedFunctions, delayed)

	return d
}

func (d *batchPut) PutItems(items ...interface{}) *batchPut {
	d.writeItems(true, items...)
	return d
}
func (d *batchPut) DeleteItems(keys ...KeyValue) *batchPut {
	a := []interface{}{}
	for _, key := range keys {
		m := map[string]interface{}{}
		appendKeyInterface(&m, d.table, key)
		a = append(a, m)
	}
	d.writeItems(false, a...)
	return d
}

func (d *batchPut) Build() (input []dynamodb.BatchWriteItemInput, err error) {
	for _, function := range d.delayedFunctions {
		if err = function(); err != nil {
			return
		}
	}
	input = d.batches
	return
}

/**
 ** ExecuteWith ... Execute a dynamo BatchWriteItem call with a passed in dynamodb instance and unprocessed item pointer function
 ** ctx - An instance of context
 ** dynamo - The underlying dynamodb api
 ** unprocessedItem - The item pointer function, which is called on each object returned from dynamodb that could not be processed.
 ** 				The function should store each item pointer in an array before returning.
 **
 */
func (d *batchPut) ExecuteWith(ctx context.Context, dynamo DynamoDBIFace, unprocessedItem func() interface{}, opts ...request.Option) error {

	batches, err := d.Build()
	if err != nil {
		return err
	}
	for _, batch := range batches {
		out, err := dynamo.BatchWriteItemWithContext(ctx, &batch, opts...)
		if err != nil {
			return handleAwsErr(err)
		}

		for _, items := range out.UnprocessedItems {
			for _, item := range items {
				err = dynamodbattribute.UnmarshalMap(item.PutRequest.Item, unprocessedItem())
				if err != nil {
					return handleAwsErr(err)
				}
			}
		}
	}

	return nil
}

/***************************************************************************************/
/*************************************** DeleteItem ************************************/
/***************************************************************************************/
type deleteItem dynamodb.DeleteItemInput

/*DeleteItem represents dynamo delete item call*/
func (table DynamoTable) DeleteItem(key KeyValue) *deleteItem {
	q := deleteItem(dynamodb.DeleteItemInput{})
	q.TableName = &table.Name
	appendKeyAttribute(&q.Key, table, key)
	return &q
}

func (d *deleteItem) ReturnAllOld() {
	(*dynamodb.DeleteItemInput)(d).SetReturnValues("ALL_OLD")
}

func (d *deleteItem) ReturnNone() {
	(*dynamodb.DeleteItemInput)(d).SetReturnValues("NONE")
}

func (d *deleteItem) SetConditionExpression(c Expression) *deleteItem {
	s, m, _ := c.construct(1, true)
	d.ConditionExpression = &s
	d.ExpressionAttributeValues, _ = dynamodbattribute.MarshalMap(m)
	return d
}

func (d *deleteItem) Build() *dynamodb.DeleteItemInput {
	r := dynamodb.DeleteItemInput(*d)
	return &r
}

/**
 ** ExecuteWith ... Execute a dynamo DeleteItem call with a passed in dynamodb instance
 ** ctx - An instance of context
 ** dynamo - The underlying dynamodb api
 **
 */
func (d *deleteItem) ExecuteWith(ctx context.Context, dynamo DynamoDBIFace, opts ...request.Option) error {
	_, err := dynamo.DeleteItemWithContext(ctx, d.Build(), opts...)
	if err != nil {
		return handleAwsErr(err)
	}
	return nil
}

/***************************************************************************************/
/*********************************** UpdateItem ****************************************/
/***************************************************************************************/
type Update struct {
	input            dynamodb.UpdateItemInput
	delayedFunctions []func() error
}

/*UpdateItem represents dynamo batch get item call*/
func (table DynamoTable) UpdateItem(key KeyValue) *Update {
	q := Update{input: dynamodb.UpdateItemInput{TableName: &table.Name}}
	appendKeyAttribute(&q.input.Key, table, key)
	return &q
}

func (d *Update) ReturnAllNew() {
	d.input.SetReturnValues("ALL_New")
}

func (d *Update) ReturnAllOld() {
	d.input.SetReturnValues("ALL_OLD")
}

func (d *Update) ReturnUpdatedNew() {
	d.input.SetReturnValues("UPDATED_NEw")
}

func (d *Update) ReturnUpdatedOld() {
	d.input.SetReturnValues("UPDATED_OLD")
}

func (d *Update) ReturnNone() {
	d.input.SetReturnValues("NONE")
}

func (d *Update) SetConditionExpression(c Expression) *Update {
	delayed := func() error {
		s, m, _ := c.construct(1, true)
		d.input.ConditionExpression = &s
		ea, err := dynamodbattribute.MarshalMap(m)
		if err != nil {
			return err
		}
		if d.input.ExpressionAttributeValues == nil {
			d.input.ExpressionAttributeValues = make(map[string]*dynamodb.AttributeValue)
		}
		for k, v := range ea {
			d.input.ExpressionAttributeValues[k] = v
		}
		return nil
	}
	d.delayedFunctions = append(d.delayedFunctions, delayed)
	return d
}

func (d *Update) SetUpdateExpression(exprs ...*UpdateExpression) *Update {
	m := make(map[string]interface{})
	ms := make(map[string]string)

	c := uint(100)
	for _, expr := range exprs {
		s, mr, nc := expr.f(c)
		c = nc
		for k, v := range mr {
			m[k] = v
		}
		if ms[expr.op] == "" {
			ms[expr.op] = s
		} else {
			ms[expr.op] += ", " + s
		}
	}

	var s string
	for k, v := range ms {
		s += k + " " + v + " "
	}

	d.input.UpdateExpression = &s
	ea, err := dynamodbattribute.MarshalMap(m)
	if err != nil {
		panic(err)
	}
	if d.input.ExpressionAttributeValues == nil {
		d.input.ExpressionAttributeValues = make(map[string]*dynamodb.AttributeValue)
	}
	for k, v := range ea {
		d.input.ExpressionAttributeValues[k] = v
	}
	return d
}

func (d *Update) Build() *dynamodb.UpdateItemInput {
	r := dynamodb.UpdateItemInput((*d).input)
	return &r
}

/**
 ** ExecuteWith ... Execute a dynamo BatchGetItem call with a passed in dynamodb instance
 ** ctx - an instance of context
 ** dynamo - The underlying dynamodb api
 **
 */
func (d *Update) ExecuteWith(ctx context.Context, dynamo DynamoDBIFace, opts ...request.Option) error {
	_, err := dynamo.UpdateItemWithContext(ctx, d.Build(), opts...)
	if err != nil {
		return handleAwsErr(err)
	}
	return nil
}

/***************************************************************************************/
/********************************************** Query **********************************/
/***************************************************************************************/
type Query struct {
	*dynamodb.QueryInput
	pageSize         *int64
	capacityHandlers []func(*dynamodb.ConsumedCapacity)
}

/*Query represents dynamo batch get item call*/
func (table DynamoTable) Query(partitionKeyCondition KeyCondition, rangeKeyCondition *KeyCondition) *Query {
	var input dynamodb.QueryInput
	q := Query{
		QueryInput: &input,
	}

	var e Expression
	if rangeKeyCondition != nil {
		e = And(partitionKeyCondition, *rangeKeyCondition)
	} else {
		e = partitionKeyCondition
	}

	s, m, _ := e.construct(0, true)
	q.TableName = &table.Name
	q.KeyConditionExpression = &s
	for k, v := range m {
		appendAttribute(&q.ExpressionAttributeValues, k, v)
	}

	return &q
}

func (d *Query) SetConsistentRead(c bool) *Query {
	(*d).ConsistentRead = &c
	return d
}
func (d *Query) SetAttributesToGet(fields []DynamoField) *Query {
	a := make([]*string, len(fields))
	for i, f := range fields {
		v := f.Name()
		a[i] = &v
	}
	(*d).AttributesToGet = a
	return d
}

func (d *Query) SetLimit(limit int) *Query {
	s := int64(limit)
	d.Limit = &s
	return d
}

func (d *Query) SetPageSize(pageSize int) *Query {
	ps := int64(pageSize)
	d.pageSize = &ps
	return d
}

func (d *Query) SetScanForward(forward bool) *Query {
	d.ScanIndexForward = &forward
	return d
}

func (d *Query) WithConsumedCapacityHandler(f func(*dynamodb.ConsumedCapacity)) *Query {
	d.ReturnConsumedCapacity = aws.String("INDEXES")
	d.capacityHandlers = append(d.capacityHandlers, f)
	return d
}

func (d *Query) SetFilterExpression(c Expression) *Query {
	s, m, _ := c.construct(1, true)
	d.FilterExpression = &s

	for k, v := range m {
		appendAttribute(&d.ExpressionAttributeValues, k, v)
	}
	return d
}

func (d *Query) SetLocalIndex(idx LocalSecondaryIndex) *Query {
	d.IndexName = &idx.Name
	return d
}

func (d *Query) SetGlobalIndex(idx GlobalSecondaryIndex) *Query {
	d.IndexName = &idx.Name
	return d
}

func (d *Query) Build() *dynamodb.QueryInput {
	r := dynamodb.QueryInput(*d.QueryInput)
	if d.pageSize != nil {
		r.Limit = d.pageSize
	}

	return &r
}

/**
 ** StreamWith ... Execute a dynamo Stream call with a passed in dynamodb instance and next item pointer
 ** ctx - An instance of context
 ** dynamo - The underlying dynamodb api
 ** nextItem - The item pointer which is copied and hydrated on every item. The function SHOULD NOT
 ** 		   store each item. It should simply return an empty struct pointer. Each of which is hydrated and pused on the
 ** 			returned channel.
 **
 */
func (d *Query) StreamWith(ctx context.Context, dynamodb DynamoDBIFace, nextItem interface{}, opts ...request.Option) (c chan interface{}, e chan error) {
	v := reflect.ValueOf(nextItem)
	t := reflect.Indirect(v).Type()

	c = make(chan interface{})
	e = make(chan error)

	go func() {
		defer close(c)
		defer close(e)

		var count int64
	Execute:
		if d.Limit != nil && count >= *d.Limit {
			return
		}
		out, err := dynamodb.QueryWithContext(ctx, d.Build(), opts...)
		if err != nil {
			e <- handleAwsErr(err)
			return
		}

		for _, item := range out.Items {
			result := reflect.New(t).Interface()
			err = dynamodbattribute.UnmarshalMap(item, result)

			if err != nil {
				e <- handleAwsErr(err)
				return
			}
			count++
			c <- result

			for _, handler := range d.capacityHandlers {
				handler(out.ConsumedCapacity)
			}

			if d.Limit != nil && count >= *d.Limit {
				return
			}
		}

		if out.LastEvaluatedKey != nil {
			d.ExclusiveStartKey = out.LastEvaluatedKey
			goto Execute
		}

		return
	}()

	return
}

func (d *Query) StreamWithChannel(ctx context.Context, dynamodb DynamoDBIFace, channel interface{}, opts ...request.Option) (errChan chan error) {
	t := reflect.TypeOf(channel).Elem()
	isPtr := t.Kind() == reflect.Ptr
	if isPtr {
		t = t.Elem()
	}
	vc := reflect.ValueOf(channel)
	errChan = make(chan error)

	go func() {
		defer vc.Close()
		defer close(errChan)

		var count int64
	Execute:
		if d.Limit != nil && count >= *d.Limit {
			return
		}
		out, err := dynamodb.QueryWithContext(ctx, d.Build(), opts...)
		if err != nil {
			errChan <- handleAwsErr(err)
			return
		}

		for _, item := range out.Items {
			result := reflect.New(t).Interface()
			err = dynamodbattribute.UnmarshalMap(item, result)

			if err != nil {
				errChan <- handleAwsErr(err)
				return
			}

			value := reflect.ValueOf(result)

			count++
			if isPtr {
				vc.Send(value)
			} else {
				vc.Send(reflect.Indirect(value))
			}

			for _, handler := range d.capacityHandlers {
				handler(out.ConsumedCapacity)
			}

			if d.Limit != nil && count >= *d.Limit {
				return
			}
		}

		if out.LastEvaluatedKey != nil {
			d.ExclusiveStartKey = out.LastEvaluatedKey
			goto Execute
		}
		return
	}()

	return
}

func (d *Query) ExecuteWith(ctx context.Context, dynamodb DynamoDBIFace, nextItem interface{}, opts ...request.Option) (items []interface{}, err error) {
	c, e := d.StreamWith(ctx, dynamodb, nextItem, opts...)
	items = []interface{}{}

STREAM:
	for {
		select {
		case item, ok := <-c:
			if !ok {
				break STREAM
			}

			items = append(items, item)

		case err = <-e:
			break STREAM
		}
	}

	return items, err
}

/***************************************************************************************/
/********************************************** Scan **********************************/
/***************************************************************************************/
type Scan struct {
	*dynamodb.ScanInput
	pageSize *int64
}

/*Scan represents dynamo scan item call*/
func (table DynamoTable) Scan() *Scan {
	var input dynamodb.ScanInput
	q := Scan{
		ScanInput: &input,
	}

	q.TableName = &table.Name
	return &q
}

func (d *Scan) SetConsistentRead(c bool) *Scan {
	(*d).ConsistentRead = &c
	return d
}
func (d *Scan) SetAttributesToGet(fields []DynamoField) *Scan {
	a := make([]*string, len(fields))
	for i, f := range fields {
		v := f.Name()
		a[i] = &v
	}
	(*d).AttributesToGet = a
	return d
}

func (d *Scan) SetLimit(limit int) *Scan {
	s := int64(limit)
	d.Limit = &s
	return d
}

func (d *Scan) SetPageSize(pageSize int) *Scan {
	ps := int64(pageSize)
	d.pageSize = &ps
	return d
}

func (d *Scan) SetFilterExpression(c Expression) *Scan {
	s, m, _ := c.construct(1, true)
	d.FilterExpression = &s

	for k, v := range m {
		appendAttribute(&d.ExpressionAttributeValues, k, v)
	}
	return d
}

func (d *Scan) SetLocalIndex(idx LocalSecondaryIndex) *Scan {
	d.IndexName = &idx.Name
	return d
}

func (d *Scan) SetGlobalIndex(idx GlobalSecondaryIndex) *Scan {
	d.IndexName = &idx.Name
	return d
}

func (d *Scan) Build() *dynamodb.ScanInput {
	r := dynamodb.ScanInput(*d.ScanInput)
	if d.pageSize != nil {
		r.Limit = d.pageSize
	}
	return &r
}

/**
 ** ExecuteWith ... Execute a dynamo Scan call with a passed in dynamodb instance and next item pointer
 ** dynamo - The underlying dynamodb api
 ** nextItem - The item pointer function, which is called on each new object returned from dynamodb. The function SHOULD NOT
 ** 		   store each item. It should simply return an empty struct pointer. Each of which is hydrated and pushed on
 ** 		   the returned channel.
 **
 */
func (d *Scan) ExecuteWith(ctx context.Context, dynamodb DynamoDBIFace, nextItem interface{}, opts ...request.Option) (c chan interface{}, e chan error) {

	c = make(chan interface{})
	e = make(chan error)

	go func() {
		defer close(c)
		defer close(e)

		var count int64
	Execute:
		if d.Limit != nil && count >= *d.Limit {
			return
		}
		out, err := dynamodb.ScanWithContext(ctx, d.Build(), opts...)
		if err != nil {
			e <- handleAwsErr(err)
			return
		}

		for _, item := range out.Items {
			err = dynamodbattribute.UnmarshalMap(item, &nextItem)

			if err != nil {
				e <- handleAwsErr(err)
				return
			}
			count++
			c <- nextItem
		}

		if out.LastEvaluatedKey != nil {
			d.ExclusiveStartKey = out.LastEvaluatedKey
			goto Execute
		}
		return
	}()

	return
}

func (d *Scan) StreamWithChannel(ctx context.Context, dynamodb DynamoDBIFace, channel interface{}, opts ...request.Option) (errChan chan error) {
	t := reflect.TypeOf(channel).Elem()
	isPtr := t.Kind() == reflect.Ptr
	if isPtr {
		t = t.Elem()
	}
	vc := reflect.ValueOf(channel)
	errChan = make(chan error)

	go func() {
		defer vc.Close()
		defer close(errChan)

		var count int64
	Execute:
		if d.Limit != nil && count >= *d.Limit {
			return
		}
		out, err := dynamodb.ScanWithContext(ctx, d.Build(), opts...)
		if err != nil {
			errChan <- handleAwsErr(err)
			return
		}

		for _, item := range out.Items {
			nextItem := reflect.New(t).Interface()
			err = dynamodbattribute.UnmarshalMap(item, nextItem)
			value := reflect.ValueOf(nextItem)
			if err != nil {
				errChan <- handleAwsErr(err)
				return
			}
			count++
			if isPtr {
				vc.Send(value)
			} else {
				vc.Send(reflect.Indirect(value))
			}
			if d.Limit != nil && count >= *d.Limit {
				return
			}
		}

		if out.LastEvaluatedKey != nil {
			d.ExclusiveStartKey = out.LastEvaluatedKey
			goto Execute
		}
		return
	}()

	return
}

/**********************************************************************************************/
/********************************************** Create Table **********************************/
/**********************************************************************************************/
type createTable dynamodb.CreateTableInput

func (table DynamoTable) CreateTable() *createTable {
	pk := table.PartitionKey.Name()
	pkt := "HASH"
	pktt := table.PartitionKey.Type()

	k := []*dynamodb.KeySchemaElement{
		&dynamodb.KeySchemaElement{
			AttributeName: &pk,
			KeyType:       &pkt,
		},
	}
	r := int64(100)
	w := int64(100)
	p := &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  &r,
		WriteCapacityUnits: &w,
	}

	a := []*dynamodb.AttributeDefinition{
		&dynamodb.AttributeDefinition{
			AttributeName: &pk,
			AttributeType: &pktt,
		},
	}

	if !table.RangeKey.IsEmpty() {
		rk := table.RangeKey.Name()
		rkt := "RANGE"
		rktt := table.RangeKey.Type()
		k = append(k, &dynamodb.KeySchemaElement{AttributeName: &rk, KeyType: &rkt})
		a = append(a, &dynamodb.AttributeDefinition{AttributeName: &rk, AttributeType: &rktt})
	}
	t := dynamodb.CreateTableInput{
		TableName:             &table.Name,
		KeySchema:             k,
		ProvisionedThroughput: p,
		AttributeDefinitions:  a,
	}
	c := createTable(t)

	// add GlobalSecondaryIndexes
	if len(table.GlobalSecondaryIndexes) > 0 {
		for _, gsi := range table.GlobalSecondaryIndexes {
			c = *c.WithGlobalSecondaryIndex(gsi)
		}
	}

	// add LocalSecondaryIndexes
	if len(table.LocalSecondaryIndexes) > 0 {
		for _, lsi := range table.LocalSecondaryIndexes {
			c = *c.WithLocalSecondaryIndex(lsi)
		}
	}

	return &c
}

func (d *createTable) WithLocalSecondaryIndex(lsi LocalSecondaryIndex) *createTable {
	// handle projection types and NonKeyAttributes
	var pt *string
	var nka []*string
	if lsi.ProjectionType == "" {
		pt = aws.String(ProjectionTypeALL)
	} else {
		// ALL, INCLUDE, KEYS_ONLY
		pt = aws.String(lsi.ProjectionType)
		if lsi.ProjectionType == ProjectionTypeINCLUDE {
			for _, key := range lsi.NonKeyAttributes {
				newAttr := &dynamodb.AttributeDefinition{
					AttributeName: aws.String(key.Name()),
					AttributeType: aws.String(key.Type()),
				}
				d.AttributeDefinitions = append(d.AttributeDefinitions, newAttr)
				nka = append(nka, aws.String(key.Name()))
			}
		}
	}

	// populate missing AttributeDefinitions
	pk := &dynamodb.AttributeDefinition{
		AttributeName: aws.String(lsi.PartitionKey.Name()),
		AttributeType: aws.String(lsi.PartitionKey.Type()),
	}
	rk := &dynamodb.AttributeDefinition{
		AttributeName: aws.String(lsi.SortKey.Name()),
		AttributeType: aws.String(lsi.SortKey.Type()),
	}
	d.AttributeDefinitions = append(d.AttributeDefinitions, pk)
	d.AttributeDefinitions = append(d.AttributeDefinitions, rk)

	// create lsi obj
	dynamoLsi := dynamodb.LocalSecondaryIndex{
		IndexName: &lsi.Name,
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(lsi.PartitionKey.Name()),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String(lsi.SortKey.Name()),
				KeyType:       aws.String("RANGE"),
			},
		},
		Projection: &dynamodb.Projection{
			ProjectionType:   pt,
			NonKeyAttributes: nka,
		},
	}

	// append lsi to *createTable
	d.LocalSecondaryIndexes = append(d.LocalSecondaryIndexes, &dynamoLsi)
	return d
}

func (d *createTable) WithGlobalSecondaryIndex(gsi GlobalSecondaryIndex) *createTable {
	// handle projection types and NonKeyAttributes
	var pt *string
	var nka []*string
	if gsi.ProjectionType == "" {
		pt = aws.String(ProjectionTypeALL)
	} else {
		// ALL, INCLUDE, KEYS_ONLY
		pt = aws.String(gsi.ProjectionType)
		if gsi.ProjectionType == ProjectionTypeINCLUDE {
			for _, key := range gsi.NonKeyAttributes {
				newAttr := &dynamodb.AttributeDefinition{
					AttributeName: aws.String(key.Name()),
					AttributeType: aws.String(key.Type()),
				}
				d.AttributeDefinitions = append(d.AttributeDefinitions, newAttr)
				nka = append(nka, aws.String(key.Name()))
			}
		}
	}

	// setup default provisioning
	var gsir *int64
	var gsiw *int64

	if gsi.ReadUnits != 0 {
		gsir = &gsi.ReadUnits
	} else {
		gsir = aws.Int64(10)
	}
	if gsi.WriteUnits != 0 {
		gsiw = &gsi.WriteUnits
	} else {
		gsiw = aws.Int64(10)
	}

	// populate missing AttributeDefinitions
	pk := &dynamodb.AttributeDefinition{
		AttributeName: aws.String(gsi.PartitionKey.Name()),
		AttributeType: aws.String(gsi.PartitionKey.Type()),
	}
	rk := &dynamodb.AttributeDefinition{
		AttributeName: aws.String(gsi.RangeKey.Name()),
		AttributeType: aws.String(gsi.RangeKey.Type()),
	}
	d.AttributeDefinitions = append(d.AttributeDefinitions, pk)
	d.AttributeDefinitions = append(d.AttributeDefinitions, rk)

	// create gsi obj
	dynamoGsi := dynamodb.GlobalSecondaryIndex{
		IndexName: &gsi.Name,
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(gsi.PartitionKey.Name()),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String(gsi.RangeKey.Name()),
				KeyType:       aws.String("RANGE"),
			},
		},
		Projection: &dynamodb.Projection{
			ProjectionType:   pt,
			NonKeyAttributes: nka,
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  gsir,
			WriteCapacityUnits: gsiw,
		},
	}

	// append gsi to *createTable
	d.GlobalSecondaryIndexes = append(d.GlobalSecondaryIndexes, &dynamoGsi)
	return d
}

func (d *createTable) Build() *dynamodb.CreateTableInput {
	r := dynamodb.CreateTableInput(*d)
	defer time.Sleep(time.Duration(500) * time.Millisecond)
	return &r
}

func (d *createTable) ExecuteWith(ctx context.Context, dynamo DynamoDBIFace, opts ...request.Option) error {
	defer time.Sleep(time.Duration(500) * time.Millisecond)
	_, err := dynamo.CreateTableWithContext(ctx, d.Build(), opts...)
	return handleAwsErr(err)
}

/**********************************************************************************************/
/********************************************** Delete Table **********************************/
/**********************************************************************************************/
type deleteTable dynamodb.DeleteTableInput

func (table DynamoTable) DeleteTable() *deleteTable {
	r := deleteTable(dynamodb.DeleteTableInput{TableName: &table.Name})
	return &r
}

func (d *deleteTable) Build() *dynamodb.DeleteTableInput {
	r := dynamodb.DeleteTableInput(*d)
	return &r
}

func (d *deleteTable) ExecuteWith(ctx context.Context, dynamo DynamoDBIFace, opts ...request.Option) error {
	defer time.Sleep(time.Duration(500) * time.Millisecond)
	_, err := dynamo.DeleteTableWithContext(ctx, d.Build(), opts...)
	return handleAwsErr(err)
}

/*****************************************   Helpers  ******************************************/
func appendKeyInterface(m *map[string]interface{}, table DynamoTable, key KeyValue) {
	if *m == nil {
		*m = map[string]interface{}{}
	}
	(*m)[table.PartitionKey.Name()] = key.PartitionKey

	if !table.RangeKey.IsEmpty() {
		(*m)[table.RangeKey.Name()] = key.RangeKey
	}

}
func appendKeyAttribute(m *map[string]*dynamodb.AttributeValue, table DynamoTable, key KeyValue) (err error) {
	err = appendAttribute(m, table.PartitionKey.Name(), key.PartitionKey)
	if err != nil {
		return
	} else if !table.RangeKey.IsEmpty() {
		err = appendAttribute(m, table.RangeKey.Name(), key.RangeKey)
		if err != nil {
			return
		}
	}
	return
}

func appendAttribute(m *map[string]*dynamodb.AttributeValue, key string, value interface{}) (err error) {
	if *m == nil {
		*m = make(map[string]*dynamodb.AttributeValue)
	}
	v, err := dynamodbattribute.Marshal(value)
	if err == nil {
		(*m)[key] = v
	}
	return
}

func handleAwsErr(err error) error {
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("Error: %v, %v", awsErr.Code(), awsErr.Message())
		}
	}

	return err
}
