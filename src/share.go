// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package drive

import (
	"fmt"
	"strings"
	"sync"

	"github.com/odeke-em/log"
)

type AccountType int

const (
	UnknownAccountType AccountType = 1 << iota
	Anyone
	User
	Domain
	Group
)

type Role int

const (
	UnknownRole Role = 1 << iota
	Owner
	Reader
	Writer
	Commenter
)

const (
	NoopOnShare = 1 << iota
	Notify
	WithLink
)

type shareChange struct {
	emailMessage string
	emails       []string
	roles        []Role
	accountTypes []AccountType
	files        []*File
	revoke       bool
	notify       bool
	// withLink turns off public file indexing so that
	// the file will only be shared to those with the link
	withLink bool
}

type permission struct {
	fileId      string
	value       string
	message     string
	role        Role
	accountType AccountType

	notify   bool
	withLink bool
}

func (r *Role) String() string {
	switch *r {
	case Owner:
		return "owner"
	case Reader:
		return "reader"
	case Writer:
		return "writer"
	case Commenter:
		return "commenter"
	}
	return "unknown"
}

func unknownRole(r string) bool {
	return strings.ToLower(r) == "unknown"
}

func (a *AccountType) String() string {
	switch *a {
	case Anyone:
		return "anyone"
	case User:
		return "user"
	case Domain:
		return "domain"
	case Group:
		return "group"
	}
	return "unknown"
}

func unknownAccountType(a string) bool {
	return strings.ToLower(a) == "unknown"
}

func stringToRole() func(string) Role {
	roleMap := make(map[string]Role)
	roles := []Role{UnknownRole, Owner, Reader, Writer, Commenter}
	for _, role := range roles {
		roleMap[role.String()] = role
	}
	return func(s string) Role {
		r, ok := roleMap[strings.ToLower(s)]
		if !ok {
			return Reader
		}
		return r
	}
}

func stringToAccountType() func(string) AccountType {
	accountMap := make(map[string]AccountType)
	accounts := []AccountType{UnknownAccountType, Anyone, User, Domain, Group}
	for _, account := range accounts {
		accountMap[account.String()] = account
	}
	return func(s string) AccountType {
		a, ok := accountMap[strings.ToLower(s)]
		if !ok {
			return User
		}
		return a
	}
}

var reverseRoleResolve = stringToRole()
var reverseAccountTypeResolve = stringToAccountType()

func reverseRolesResolver(roleArgv ...string) (roles []Role) {
	alreadySeen := map[Role]bool{}
	for _, roleStr := range roleArgv {
		role := reverseRoleResolve(roleStr)
		if _, seen := alreadySeen[role]; !seen {
			roles = append(roles, role)
			alreadySeen[role] = true
		}
	}

	return roles
}

func reverseAccountTypesResolver(accArgv ...string) (accountTypes []AccountType) {
	for _, accStr := range accArgv {
		accountTypes = append(accountTypes, reverseAccountTypeResolve(accStr))
	}

	return accountTypes
}

func (g *Commands) resolveRemotePaths(relToRootPaths []string, byId bool) (files []*File) {
	var wg sync.WaitGroup

	resolver := g.rem.FindByPath
	if byId {
		resolver = g.rem.FindById
	}

	wg.Add(len(relToRootPaths))
	for _, relToRoot := range relToRootPaths {
		go func(p string, wgg *sync.WaitGroup) {
			defer wgg.Done()
			file, err := resolver(p)
			if err != nil || file == nil {
				return
			}
			files = append(files, file)
		}(relToRoot, &wg)
	}
	wg.Wait()
	return files
}

func emailsToIds(g *Commands, emails []string) map[string]string {
	emailToIds := make(map[string]string)
	var wg sync.WaitGroup
	wg.Add(len(emails))
	for _, email := range emails {
		go func(email string, wgg *sync.WaitGroup) {
			defer wgg.Done()
			emailId, err := g.rem.idForEmail(email)
			if err == nil {
				emailToIds[email] = emailId
			}
		}(email, &wg)
	}
	wg.Wait()
	return emailToIds
}

func (c *Commands) Unshare(byId bool) (err error) {
	return c.share(true, byId)
}

func (c *Commands) Share(byId bool) (err error) {
	return c.share(false, byId)
}

func showPromptShareChanges(logy *log.Logger, change *shareChange) Agreement {
	if len(change.files) < 1 {
		return NotApplicable
	}

	verb := "unshare"
	shareInfo := "Revoke access"
	extraShareInfo := ""
	if !change.revoke {
		shareInfo = "Provide access"
		if len(change.emails) < 1 {
			return NotApplicable
		}

		verb = "share"
		if change.notify && len(change.emailMessage) >= 1 {
			extraShareInfo = fmt.Sprintf("Message:\n\t\033[33m%s\033[00m\n", change.emailMessage)
		}
	}

	if len(change.accountTypes) >= 1 {
		logy.Logf("%s for accountType(s)\n", shareInfo)
		for _, accountType := range change.accountTypes {
			logy.Logf("\t\033[32m%s\033[00m\n", accountType.String())
		}

		logy.Logln(extraShareInfo)
	}

	if len(change.roles) >= 1 {
		logy.Logln("For roles(s)")
		for _, role := range change.roles {
			logy.Logf("\t\033[33m%s\033[00m\n", role.String())
		}
	}

	if len(change.emails) >= 1 {
		logy.Logf("\nAddressees:\n")
		for _, email := range change.emails {
			if email == "" {
				email = "Anyone with the link"
			}
			logy.Logf("\t\033[92m+\033[00m %s\n", email)
		}
	}

	logy.Logf("\nFile(s) to %s:\n", verb)
	for _, file := range change.files {
		if file == nil {
			continue
		}
		logy.Logf("\t\033[92m+\033[00m %s\n", file.Name)
	}

	logy.Logln()
	return promptForChanges()
}

func (c *Commands) playShareChanges(change *shareChange) (err error) {
	if c.opts.canPrompt() {
		if status := showPromptShareChanges(c.log, change); !accepted(status) {
			return status.Error()
		}
	}

	fnName := "unshare"
	fn := c.rem.revokePermissions

	if !change.revoke {
		fnName = "share"
		fn = func(perm *permission) error {
			_, err := c.rem.insertPermissions(perm)
			return err
		}
	}

	successes := 0

	for _, file := range change.files {
		for _, accountType := range change.accountTypes {
			for _, email := range change.emails {
				if accountType == Anyone && email != "" {
					// It doesn't make sense to share a file with
					// permission "anyone" yet provide an email
					continue
				}

				for _, role := range change.roles {
					perm := permission{
						fileId:      file.Id,
						value:       email,
						role:        role,
						accountType: accountType,

						notify:   change.notify,
						message:  change.emailMessage,
						withLink: change.withLink,
					}

					if ferr := fn(&perm); ferr != nil {
						err = reComposeError(err, fmt.Sprintf("%s err %s: %v\n", fnName, file.Name, ferr))
					} else {
						successes += 1
						if c.opts.Verbose {
							c.log.Logf("successful %s for %s with email %q, role %q accountType %q\n",
								fnName, file.Name, email, role.String(), accountType.String())
						}
					}
				}
			}
		}
	}

	if err != nil {
		return err
	}

	if successes < 1 {
		return noMatchesFoundErr(fmt.Errorf("no matches found!"))
	}

	return nil
}

func (c *Commands) share(revoke, byId bool) (err error) {
	files := c.resolveRemotePaths(c.opts.Sources, byId)

	var emails []string
	var emailMessage string

	roles := []Role{}

	// By default, if they don't specify any
	// account types to invoke, let's assume User.
	// See Issue https://github.com/odeke-em/drive/issues/747.
	accountTypes := []AccountType{User}

	if revoke {
		// In case of unsharing, when a user doesn't specify the
		// roles, the addressee should be removed from all roles
		roles = append(roles, Reader, Writer, Commenter)
	} else {
		roles = append(roles, Reader)
	}

	meta := *c.opts.Meta

	if meta != nil {
		emailList, eOk := meta[EmailsKey]
		if eOk {
			emails = emailList
			if false {
				emailIdMap := emailsToIds(c, emailList)
				c.log.Logln(emailIdMap)
			}
		}

		roleList, rOk := meta[RoleKey]
		if rOk && len(roleList) >= 1 {
			roles = reverseRolesResolver(roleList...)
		}

		accountTypeList, aOk := meta[AccountTypeKey]
		if aOk && len(accountTypeList) >= 1 {
			accountTypes = reverseAccountTypesResolver(accountTypeList...)
		}

		emailMessageList, emOk := meta[EmailMessageKey]
		if emOk && len(emailMessageList) >= 1 {
			emailMessage = strings.Join(emailMessageList, "\n")
		}
	}

	if revoke && len(emails) < 1 {
		// Account for case for unshare where certain types do not include the emails
		// e.g revoking access to the entire domain, or user groups yet no email
		// can be found
		emails = append(emails, "")
	}

	anyoneAlreadySet := false
	// Now here we have to match up emails with roles
	// See Issue #568. If we've requested say `anyone`, this role needs no email
	// address, so we should add an empty "" to the list of emails
	for _, accountType := range accountTypes {
		if accountType == Anyone {
			anyoneAlreadySet = true
			emails = append(emails, "")
		}
	}

	change := shareChange{
		files:  files,
		revoke: revoke,
		roles:  roles,
		emails: emails,

		accountTypes: accountTypes,
		emailMessage: emailMessage,

		notify:   (c.opts.TypeMask & Notify) == Notify,
		withLink: (c.opts.TypeMask & WithLink) == WithLink,
	}

	if len(change.emails) < 1 && change.withLink { // They've basically requested a share to everyone but with link
		change.emails = append(change.emails, "")
		if !anyoneAlreadySet {
			change.accountTypes = append(change.accountTypes, Anyone)
		}
	}

	return c.playShareChanges(&change)
}
