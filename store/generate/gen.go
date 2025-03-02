//go:generate go run gen.go
package main

import (
	"path/filepath"
	"runtime"

	"github.com/Nicknamezz00/org-invitation-autobot/store"
	"github.com/spf13/viper"
	"gorm.io/gen"
)

func main() {
	db := store.New(viper.GetViper())

	_, filePath, _, _ := runtime.Caller(0)
	absolutePath, _ := filepath.Abs(filePath)

	outPath := filepath.Dir(absolutePath)
	g := gen.NewGenerator(gen.Config{
		OutPath:           filepath.Join(outPath, "query"),
		ModelPkgPath:      filepath.Join(outPath, "model"),
		Mode:              gen.WithQueryInterface | gen.WithDefaultQuery,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})
	g.UseDB(db)
	g.ApplyBasic(
		g.GenerateModelAs("auto_org_invitation.invitations", "InvitationModel"),
		g.GenerateModelAs("auto_org_invitation.failed_invitations", "FailedInvitationModel"),
		g.GenerateModelAs("auto_org_invitation.successful_invitations", "SuccessfulInvitationModel"),
	)
	g.Execute()
}
