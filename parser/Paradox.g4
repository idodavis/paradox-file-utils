grammar Paradox;

file: BOM? line* EOF;

line: statement | COMMENT | WS;

statement: assignment;

assignment: key WS* operator WS* value;

key:
	ID // e.g. is_cancel_option
	| ID (COLON ID)? (DOT ID)? // e.g. birth.1001, scope:child, namespace
	| ID WS+ ID; // e.g. scripted_trigger allow_naming_on_birth_of_dynasty_child_trigger

value: block | valueAtom (WS* operator WS* valueAtom)?;

valueAtom: NUMBER | STRING | ID;

block: LBRACE blockContent* RBRACE;

blockContent: statement | value | COMMENT | WS;

operator: EQ | LT | GT | LTE | GTE | NEQ | QEQ;

ID: [\p{L}\p{N}_.$:]+;
NUMBER: '-'? [0-9]+ ('.' [0-9]+)?;
STRING: '"' (~["\r\n])* '"';

// Operators
EQ: '=';
LT: '<';
GT: '>';
LTE: '<=';
GTE: '>=';
NEQ: '!=';
QEQ: '?=';

LBRACE: '{';
RBRACE: '}';
COLON: ':';
DOT: '.';

BOM:
	'\uFEFF'; //Because paradox is crazy and requires BOM in their files
COMMENT: '#' ~[\r\n]*;
WS: [ \t\r\n]+;