// Code generated from parser/Paradox.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // Paradox

import "github.com/antlr4-go/antlr/v4"

// ParadoxListener is a complete listener for a parse tree produced by ParadoxParser.
type ParadoxListener interface {
	antlr.ParseTreeListener

	// EnterFile is called when entering the file production.
	EnterFile(c *FileContext)

	// EnterLine is called when entering the line production.
	EnterLine(c *LineContext)

	// EnterStatement is called when entering the statement production.
	EnterStatement(c *StatementContext)

	// EnterAssignment is called when entering the assignment production.
	EnterAssignment(c *AssignmentContext)

	// EnterKey is called when entering the key production.
	EnterKey(c *KeyContext)

	// EnterValue is called when entering the value production.
	EnterValue(c *ValueContext)

	// EnterValueAtom is called when entering the valueAtom production.
	EnterValueAtom(c *ValueAtomContext)

	// EnterBlock is called when entering the block production.
	EnterBlock(c *BlockContext)

	// EnterBlockContent is called when entering the blockContent production.
	EnterBlockContent(c *BlockContentContext)

	// EnterOperator is called when entering the operator production.
	EnterOperator(c *OperatorContext)

	// ExitFile is called when exiting the file production.
	ExitFile(c *FileContext)

	// ExitLine is called when exiting the line production.
	ExitLine(c *LineContext)

	// ExitStatement is called when exiting the statement production.
	ExitStatement(c *StatementContext)

	// ExitAssignment is called when exiting the assignment production.
	ExitAssignment(c *AssignmentContext)

	// ExitKey is called when exiting the key production.
	ExitKey(c *KeyContext)

	// ExitValue is called when exiting the value production.
	ExitValue(c *ValueContext)

	// ExitValueAtom is called when exiting the valueAtom production.
	ExitValueAtom(c *ValueAtomContext)

	// ExitBlock is called when exiting the block production.
	ExitBlock(c *BlockContext)

	// ExitBlockContent is called when exiting the blockContent production.
	ExitBlockContent(c *BlockContentContext)

	// ExitOperator is called when exiting the operator production.
	ExitOperator(c *OperatorContext)
}
