package msgpack

// data types
// https://github.com/msgpack/msgpack/blob/master/spec.md
const (
	Map15LengthMask byte = 0x80
	Map16                = 0xde
	Map32                = 0xdf

	Array15LengthMask byte = 0x90

	String31LengthMask byte = 0xa0

	Bin8 byte = 0xc4

	Uint8  byte = 0xcc
	Uint16      = 0xcd
	Uint32      = 0xce
	Uint64      = 0xcf
	Int8        = 0xd0
	Int16       = 0xd1
	Int32       = 0xd2
	Int64       = 0xd3
)

// https://github.com/tarantool/tarantool/blob/1.9/src/box/iproto_constants.h

//-- пользовательские ключи
//<iproto_sync>          ::= 0x01
//<iproto_schema_id>     ::= 0x05  /* также schema_version */
//<iproto_space_id>      ::= 0x10
//<iproto_index_id>      ::= 0x11
//<iproto_limit>         ::= 0x12
//<iproto_offset>        ::= 0x13
//<iproto_iterator>      ::= 0x14
//<iproto_key>           ::= 0x20
//<iproto_tuple>         ::= 0x21
//<iproto_function_name> ::= 0x22
//<iproto_username>      ::= 0x23
//<iproto_expr>          ::= 0x27 /* также expression */
//<iproto_ops>           ::= 0x28
//<iproto_data>          ::= 0x30
//<iproto_error>         ::= 0x31

const (
	KeyHeader   byte = 0x00
	KeySync          = 0x01
	KeyTuple         = 0x21
	KeyUserName      = 0x23
	KeyData          = 0x30
	KeyError         = 0x31
)

//-- -- Значение ключа <code> в запросе может быть следующим:
//-- Ключи для команд пользователя
//<iproto_select>       ::= 0x01
//<iproto_insert>       ::= 0x02
//<iproto_replace>      ::= 0x03
//<iproto_update>       ::= 0x04
//<iproto_delete>       ::= 0x05
//<iproto_call_16>      ::= 0x06 /* as used in version 1.6 */
//<iproto_auth>         ::= 0x07
//<iproto_eval>         ::= 0x08
//<iproto_upsert>       ::= 0x09
//<iproto_call>         ::= 0x0a
//-- Коды для команд администратора
//-- (включая коды для инициализации набора реплик и выбора мастера)
//<iproto_ping>         ::= 0x40
//<iproto_join>         ::= 0x41 /* i.e. replication join */
//<iproto_subscribe>    ::= 0x42
//<iproto_request_vote> ::= 0x43
//
//-- -- Значение для ключа <code> в ответе может быть следующим:
//<iproto_ok>           ::= 0x00
//<iproto_type_error>   ::= 0x8XXX /* где XXX -- это значение в errcode.h */
