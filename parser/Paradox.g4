grammar Paradox;

file: BOM? line* EOF;

line: statement | COMMENT | WS;

statement: assignment;

assignment: key WS* operator WS* value;

key:
	ID // e.g. is_cancel_option
	| VARIABLE // e.g. @my_variable
	| ID (COLON ID)? (DOT ID)? // e.g. birth.1001, scope:child, namespace
	| ID WS+ ID; // e.g. scripted_trigger allow_naming_on_birth_of_dynasty_child_trigger

value:
	color_block
	| block
	| valueAtom (WS* operator WS* valueAtom)?;

valueAtom: NUMBER | STRING | ID | VARIABLE;

color_block: hsv_block | rgb_or_rgba_block;

hsv_block:
	HSV WS* LBRACE WS* NUMBER WS+ NUMBER WS+ NUMBER WS* RBRACE;

rgb_or_rgba_block:
	LBRACE WS* NUMBER WS+ NUMBER WS+ NUMBER (WS+ NUMBER)? WS* RBRACE;

block: LBRACE blockContent* RBRACE;

blockContent: statement | value | COMMENT | WS;

operator: EQ | LT | GT | LTE | GTE | NEQ | QEQ;

ID: [\p{L}\p{N}_.$:'-]+;
NUMBER: '-'? [0-9]+ ('.' [0-9]+)?;
STRING: '"' (~["\r\n])* '"';
VARIABLE: '@' ID;

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
HSV: 'hsv';

BOM:
	'\uFEFF'; //Because paradox is crazy and requires BOM in their files
COMMENT: '#' ~[\r\n]*;
WS: [ \t\r\n]+;