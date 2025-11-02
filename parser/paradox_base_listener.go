// Code generated from parser/Paradox.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // Paradox

import "github.com/antlr4-go/antlr/v4"

// BaseParadoxListener is a complete listener for a parse tree produced by ParadoxParser.
type BaseParadoxListener struct{}

var _ ParadoxListener = &BaseParadoxListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseParadoxListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseParadoxListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseParadoxListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseParadoxListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterFile is called when production file is entered.
func (s *BaseParadoxListener) EnterFile(ctx *FileContext) {}

// ExitFile is called when production file is exited.
func (s *BaseParadoxListener) ExitFile(ctx *FileContext) {}

// EnterLine is called when production line is entered.
func (s *BaseParadoxListener) EnterLine(ctx *LineContext) {}

// ExitLine is called when production line is exited.
func (s *BaseParadoxListener) ExitLine(ctx *LineContext) {}

// EnterStatement is called when production statement is entered.
func (s *BaseParadoxListener) EnterStatement(ctx *StatementContext) {}

// ExitStatement is called when production statement is exited.
func (s *BaseParadoxListener) ExitStatement(ctx *StatementContext) {}

// EnterAssignment is called when production assignment is entered.
func (s *BaseParadoxListener) EnterAssignment(ctx *AssignmentContext) {}

// ExitAssignment is called when production assignment is exited.
func (s *BaseParadoxListener) ExitAssignment(ctx *AssignmentContext) {}

// EnterKey is called when production key is entered.
func (s *BaseParadoxListener) EnterKey(ctx *KeyContext) {}

// ExitKey is called when production key is exited.
func (s *BaseParadoxListener) ExitKey(ctx *KeyContext) {}

// EnterValue is called when production value is entered.
func (s *BaseParadoxListener) EnterValue(ctx *ValueContext) {}

// ExitValue is called when production value is exited.
func (s *BaseParadoxListener) ExitValue(ctx *ValueContext) {}

// EnterValueAtom is called when production valueAtom is entered.
func (s *BaseParadoxListener) EnterValueAtom(ctx *ValueAtomContext) {}

// ExitValueAtom is called when production valueAtom is exited.
func (s *BaseParadoxListener) ExitValueAtom(ctx *ValueAtomContext) {}

// EnterBlock is called when production block is entered.
func (s *BaseParadoxListener) EnterBlock(ctx *BlockContext) {}

// ExitBlock is called when production block is exited.
func (s *BaseParadoxListener) ExitBlock(ctx *BlockContext) {}

// EnterBlockContent is called when production blockContent is entered.
func (s *BaseParadoxListener) EnterBlockContent(ctx *BlockContentContext) {}

// ExitBlockContent is called when production blockContent is exited.
func (s *BaseParadoxListener) ExitBlockContent(ctx *BlockContentContext) {}

// EnterOperator is called when production operator is entered.
func (s *BaseParadoxListener) EnterOperator(ctx *OperatorContext) {}

// ExitOperator is called when production operator is exited.
func (s *BaseParadoxListener) ExitOperator(ctx *OperatorContext) {}
