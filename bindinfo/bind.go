// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package bindinfo

import (
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
)

// HintsSet contains all hints of a query.
type HintsSet struct {
	tableHints [][]*ast.TableOptimizerHint // Slice offset is the traversal order of `SelectStmt` in the ast.
	indexHints [][]*ast.IndexHint          // Slice offset is the traversal order of `TableName` in the ast.
}

// GetFirstTableHints gets the first table hints.
func (hs *HintsSet) GetFirstTableHints() []*ast.TableOptimizerHint {
	if len(hs.tableHints) > 0 {
		return hs.tableHints[0]
	}
	return nil
}

// ContainTableHint means check whether the table hint set contain a hint.
func (hs *HintsSet) ContainTableHint(hint string) bool {
	for _, tableHintsForBlock := range hs.tableHints {
		for _, tableHint := range tableHintsForBlock {
			if tableHint.HintName.String() == hint {
				return true
			}
		}
	}
	return false
}

type hintProcessor struct {
	*HintsSet
	// bindHint2Ast indicates the behavior of the processor, `true` for bind hint to ast, `false` for extract hint from ast.
	bindHint2Ast bool
	tableCounter int
	indexCounter int
}

func (hp *hintProcessor) Enter(in ast.Node) (ast.Node, bool) {
	switch v := in.(type) {
	case *ast.SelectStmt:
		if hp.bindHint2Ast {
			if hp.tableCounter < len(hp.tableHints) {
				v.TableHints = hp.tableHints[hp.tableCounter]
			} else {
				v.TableHints = nil
			}
			hp.tableCounter++
		} else {
			hp.tableHints = append(hp.tableHints, v.TableHints)
		}
	case *ast.TableName:
		if hp.bindHint2Ast {
			if hp.indexCounter < len(hp.indexHints) {
				v.IndexHints = hp.indexHints[hp.indexCounter]
			} else {
				v.IndexHints = nil
			}
			hp.indexCounter++
		} else {
			hp.indexHints = append(hp.indexHints, v.IndexHints)
		}
	}
	return in, false
}

func (hp *hintProcessor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

// CollectHint collects hints for a statement.
func CollectHint(in ast.StmtNode) *HintsSet {
	hp := hintProcessor{HintsSet: &HintsSet{tableHints: make([][]*ast.TableOptimizerHint, 0, 4), indexHints: make([][]*ast.IndexHint, 0, 4)}}
	in.Accept(&hp)
	return hp.HintsSet
}

// BindHint will add hints for stmt according to the hints in `hintsSet`.
func BindHint(stmt ast.StmtNode, hintsSet *HintsSet) ast.StmtNode {
	hp := hintProcessor{HintsSet: hintsSet, bindHint2Ast: true}
	stmt.Accept(&hp)
	return stmt
}

// ParseHintsSet parses a SQL string and collect HintsSet.
func ParseHintsSet(p *parser.Parser, sql, charset, collation string) (*HintsSet, error) {
	stmtNode, err := p.ParseOneStmt(sql, charset, collation)
	if err != nil {
		return nil, err
	}
	return CollectHint(stmtNode), nil
}
