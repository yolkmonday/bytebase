package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"

	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatability.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (adv *CompatibilityAdvisor) Check(ctx advisor.AdvisorContext, statement string) ([]advisor.Advice, error) {
	p := parser.New()

	root, _, err := p.Parse(statement, ctx.Charset, ctx.Collation)
	if err != nil {
		return []advisor.Advice{
			{
				Status:  advisor.Error,
				Title:   "Syntax error",
				Content: err.Error(),
			},
		}, nil
	}

	c := &compatibilityChecker{}
	for _, stmtNode := range root {
		(stmtNode).Accept(c)
	}

	if len(c.advisorList) == 0 {
		c.advisorList = append(c.advisorList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "Migration is backward compatible"})
	}
	return c.advisorList, nil
}

type compatibilityChecker struct {
	advisorList []advisor.Advice
}

func (v *compatibilityChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := common.Ok
	switch node := in.(type) {
	// DROP DATABASE
	case *ast.DropDatabaseStmt:
		code = common.CompatibilityDropDatabase
	// RENAME TABLE
	case *ast.RenameTableStmt:
		code = common.CompatibilityRenameTable
	// DROP TABLE/VIEW
	case *ast.DropTableStmt:
		code = common.CompatibilityDropTable
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			fmt.Printf("spec %d: %+v\n\n", spec.Tp, spec)
			// RENAME COLUMN
			if spec.Tp == ast.AlterTableRenameColumn {
				code = common.CompatibilityRenameColumn
				break
			}
			// DROP COLUMN
			if spec.Tp == ast.AlterTableDropColumn {
				code = common.CompatibilityDropColumn
				break
			}

			if spec.Tp == ast.AlterTableAddConstraint {
				// ADD PRIMARY KEY
				if spec.Constraint.Tp == ast.ConstraintPrimaryKey {
					code = common.CompatibilityAddPrimaryKey
					break
				}
				// ADD UNIQUE/UNIQUE KEY/UNIQUE INDEX
				if spec.Constraint.Tp == ast.ConstraintPrimaryKey ||
					spec.Constraint.Tp == ast.ConstraintUniq ||
					spec.Constraint.Tp == ast.ConstraintUniqKey {
					code = common.CompatibilityAddUniqueKey
					break
				}
				// ADD FOREIGN KEY
				if spec.Constraint.Tp == ast.ConstraintForeignKey {
					code = common.CompatibilityAddForeignKey
					break
				}
				// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
				// ADD CHECK ENFORCED
				if spec.Constraint.Tp == ast.ConstraintCheck && spec.Constraint.Enforced {
					code = common.CompatibilityAddCheck
					break
				}
			}

			// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
			// ALTER CHECK ENFORCED
			if spec.Tp == ast.AlterTableAlterCheck {
				if spec.Constraint.Enforced {
					code = common.CompatibilityAlterCheck
					break
				}
			}

			// MODIFY COLUMN / CHANGE COLUMN
			// Due to the limitation that we don't know the current data type of the column before the change,
			// so we treat all as incompatible. This generates false positive when:
			// 1. Change to a compatible data type such as INT to BIGINT
			// 2. Change property like comment, change it to NULL
			if spec.Tp == ast.AlterTableModifyColumn || spec.Tp == ast.AlterTableChangeColumn {
				code = common.CompatibilityAlterColumn
				break
			}
		}

	// ALTER VIEW TBD: https://github.com/pingcap/parser/pull/1252
	// case *ast.AlterViewStmt:

	// CREATE UNIQUE INDEX
	case *ast.CreateIndexStmt:
		if node.KeyType == ast.IndexKeyTypeUnique {
			code = common.CompatibilityAddUniqueKey
		}
	}

	if code != common.Ok {
		v.advisorList = append(v.advisorList, advisor.Advice{
			Status:  advisor.Warn,
			Code:    code,
			Title:   "Potential incompatible migration",
			Content: fmt.Sprintf("%q may cause incompatibility with the existing data and code", in.Text()),
		})
	}
	return in, false
}

func (v *compatibilityChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
