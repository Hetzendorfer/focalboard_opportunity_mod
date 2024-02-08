package api

import (
	// "encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/focalboard/server/services/audit"
	mmModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
)

const (
	archiveExtension = ".boardarchive"
)

func (a *API) registerAchivesRoutes(r *mux.Router) {
	// Archive APIs
	r.HandleFunc("/boards/{boardID}/archive/export", a.sessionRequired(a.handleArchiveExportBoard)).Methods("GET")
	r.HandleFunc("/boards/{boardID}/archive/import/opportunities", a.sessionRequired(a.handleArchiveImportOpportunities)).Methods("POST")
	r.HandleFunc("/teams/{teamID}/archive/import", a.sessionRequired(a.handleArchiveImport)).Methods("POST")
	r.HandleFunc("/teams/{teamID}/archive/export", a.sessionRequired(a.handleArchiveExportTeam)).Methods("GET")
}

func (a *API) handleArchiveExportBoard(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /boards/{boardID}/archive/export archiveExportBoard
	//
	// Exports an archive of all blocks for one boards.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: boardID
	//   in: path
	//   description: Id of board to export
	//   required: true
	//   type: string
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     content:
	//       application-octet-stream:
	//         type: string
	//         format: binary
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	vars := mux.Vars(r)
	boardID := vars["boardID"]
	userID := getUserID(r)

	// check user has permission to board
	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionViewBoard) {
		// if this user has `manage_system` permission and there is a license with the compliance
		// feature enabled, then we will allow the export.
		license := a.app.GetLicense()
		if !a.permissions.HasPermissionTo(userID, mmModel.PermissionManageSystem) || license == nil || !(*license.Features.Compliance) {
			a.errorResponse(w, r, model.NewErrPermission("access denied to board"))
			return
		}
	}

	auditRec := a.makeAuditRecord(r, "archiveExportBoard", audit.Fail)
	defer a.audit.LogRecord(audit.LevelRead, auditRec)
	auditRec.AddMeta("BoardID", boardID)

	board, err := a.app.GetBoard(boardID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	opts := model.ExportArchiveOptions{
		TeamID:   board.TeamID,
		BoardIDs: []string{board.ID},
	}

	filename := fmt.Sprintf("archive-%s%s", time.Now().Format("2006-01-02"), archiveExtension)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Transfer-Encoding", "binary")

	if err := a.app.ExportArchive(w, opts); err != nil {
		a.errorResponse(w, r, err)
	}

	auditRec.Success()
}

func (a *API) filterBlocksByOpportunityId(blocks []*model.Block, opportunities *[]Opportunity, boardTemplate *model.Board, templateBlockID string, userID string) []*model.Block {
	var updatedBlocks []*model.Block

	boardID := boardTemplate.ID
	possibleValues := boardTemplate.CardProperties

	for _, opportunity := range *opportunities {
		var found model.Block = model.Block{Title: ""}

		for _, block := range blocks {
			if opportunity.OpportunityId == block.Fields["properties"].(map[string]interface{})[opportunityIdKey] {
				found = *block
				break
			}
		}

		if found.Title != "" {
			duplicatedBlock := found
			convertOpportunityToBlock(opportunity, &duplicatedBlock, possibleValues)
			updatedBlocks = append(updatedBlocks, &duplicatedBlock)
		} else {
			fmt.Println("Duplicating block else statement")
			duplicateBlocks, err := a.app.DuplicateBlock(boardID, templateBlockID, userID, true)
			if err != nil {
			}

			duplicatedBlock := duplicateBlocks[0]
			convertOpportunityToBlock(opportunity, duplicatedBlock, possibleValues)
			updatedBlocks = append(updatedBlocks, duplicatedBlock)
		}
	}

	return updatedBlocks
}

type ReturnTypeABC struct {
	opportunityId []string
	blocks        []*model.Block
}

func (a *API) handleArchiveImportOpportunities(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /boards/{boardID}/archive/import/opportunities
	//
	//     post:
	//         consumes:
	//             - multipart/form-data
	//         operationId: importOpportunitiers
	//         parameters:
	//             - description: Team ID
	//               in: path
	//               name: teamID
	//               required: true
	//               type: string
	//             - description: xlsx file to import
	//               in: formData
	//               name: file
	//               required: true
	//               type: file
	//         produces:
	//             - application/json
	//         responses:
	//             "200":
	//                 description: success
	//             default:
	//                 description: internal error
	//                 schema:
	//                     $ref: '#/definitions/ErrorResponse'
	//         security:
	//             - BearerAuth: []
	//         summary: Import opportunities to all boards in a team.

	vars := mux.Vars(r)
	boardID := vars["boardID"]
	userID := getUserID(r)

	_ = boardID

	// check user has permission to board
	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionViewBoard) {
		// if this user has `manage_system` permission and there is a license with the compliance
		// feature enabled, then we will allow the export.
		license := a.app.GetLicense()
		if !a.permissions.HasPermissionTo(userID, mmModel.PermissionManageSystem) || license == nil || !(*license.Features.Compliance) {
			a.errorResponse(w, r, model.NewErrPermission("access denied to board"))
			return
		}
	}

	isGuest, err := a.userIsGuest(userID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if isGuest {
		a.errorResponse(w, r, model.NewErrPermission("access denied to create board"))
		return
	}

	// retrieve boards list
	// boards, err := a.app.GetBoardsForUserAndTeam(userID, teamID, !isGuest)
	// if err != nil {
	// 	a.errorResponse(w, r, err)
	// 	return
	// }

	// start read file from request
	file, _, err := r.FormFile(UploadFormFileKey)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}

	var opportunitesFromFile []Opportunity
	readOpportunitiesFromFile(file, &opportunitesFromFile)
	// end read file from request

	// blocks that will be updated
	var allUserBlocks []*model.Block
	var userCardBlocks []*model.Block

	// retrieve all existing blocks from the board
	allUserBlocks, err = a.app.GetBlocks(boardID, "", "card")
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	templateBlockID := "noch keine ID gefunden"

	for i := 0; i < len(allUserBlocks); i++ {
		if allUserBlocks[i].Fields["isTemplate"] == true && allUserBlocks[i].Title == "Neue Ausschreibung" {
			templateBlockID = allUserBlocks[i].ID
			break
		}
	}

	for i := 0; i < len(allUserBlocks); i++ {
		if allUserBlocks[i].Fields["isTemplate"] == false {
			userCardBlocks = append(userCardBlocks, allUserBlocks[i])
		}
	}

	// opportunitesFromFile = opportunitesFromFile[:5]

	boardTemplate, err := a.app.GetBoard(boardID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	// Check if the board contains all the master client names needed for opportunities
	// masterClientNameOptions := selectPossibleValueById(masterClientNameKey, boardTemplate.CardProperties)
	// fmt.Println("Length of Master Client Name Options: ", len(masterClientNameOptions))
	// fmt.Println("Master Client Name Options: ", masterClientNameOptions)

	// for _, opportunity := range opportunitesFromFile {
	// 	var newOptions []map[string]interface{}
	// 	newOptions = searchForNearestPossibleOptionAndModifyIfItDoesntExist(opportunity.MasterClientName, masterClientNameOptions)

	// 	masterClientNameOptions = newOptions
	// }

	// fmt.Println("Length of new Optionen: ", len(masterClientNameOptions))
	// fmt.Println("Neue Optionen: ", masterClientNameOptions)

	// var patchBoard *model.BoardPatch = &model.BoardPatch{
	// 	UpdatedCardProperties: []map[string]interface{}{
	// 		{
	// 			"id":      masterClientNameKey,
	// 			"name":    "Kunde",
	// 			"options": masterClientNameOptions,
	// 			"type":    "select",
	// 		},
	// 	},
	// }

	// a.app.PatchBoard(patchBoard, boardID, userID)
	// newBoardTemplate, err := a.app.GetBoard(boardID)
	// if err != nil {
	// 	a.errorResponse(w, r, err)
	// 	return
	// }

	// boardTemplate = newBoardTemplate

	_ = templateBlockID

	var updatedBlocks []*model.Block
	updatedBlocks = a.filterBlocksByOpportunityId(userCardBlocks, &opportunitesFromFile, boardTemplate, templateBlockID, userID)

	tempVar, err := a.app.InsertBlocksAndNotify(updatedBlocks, userID, true)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	} else {

	}

	_ = tempVar

	// if templateBlockID != "noch keine ID gefunden" {
	// 	for i := 0; i < len(opportunitesFromFile); i++ {
	// 		duplicateBlocks, err := a.app.DuplicateBlock(boardID, templateBlockID, userID, true)
	// 		if err != nil {
	// 			return
	// 		}

	// 		_ = duplicateBlocks

	// 		var singleDuplicateBlock *model.Block
	// 		singleDuplicateBlock = duplicateBlocks[0]
	// 		convertOpportunityToBlock(opportunitesFromFile[i], singleDuplicateBlock)

	// 		blocks = append(blocks, singleDuplicateBlock)
	// 	}
	// }

	// fmt.Println(blocks)

	// tempVar, err := a.app.InsertBlocksAndNotify(blocks, userID, true)
	// if err != nil {
	// 	a.errorResponse(w, r, err)
	// 	return
	// }

	// _ = tempVar

	// auditRec := a.makeAuditRecord(r, "import", audit.Fail)
	// defer a.audit.LogRecord(audit.LevelModify, auditRec)
	// auditRec.AddMeta("filename", handle.Filename)
	// auditRec.AddMeta("size", handle.Size)

	// opt := model.ImportArchiveOptions{
	// 	TeamID:     teamID,
	// 	ModifiedBy: userID,
	// }

	// if err := a.app.ImportArchive(file, opt); err != nil {
	// 	a.logger.Debug("Error importing archive",
	// 		mlog.String("team_id", teamID),
	// 		mlog.Err(err),
	// 	)
	// 	a.errorResponse(w, r, err)
	// 	return
	// }

	// abc := ReturnTypeABC{opportunityId: opportunityIdArr, blocks: blocks}

	// fmt.Println(abc.blocks, abc.opportunityId)

	// var bErr error
	// blocks, bErr = a.app.ApplyCloudLimits(blocks)
	// if bErr != nil {
	// 	a.errorResponse(w, r, err)
	// 	return
	// }

	// json, err := json.Marshal(blocks)
	// if err != nil {
	// 	a.errorResponse(w, r, err)
	// 	return
	// }

	// jsonBytesResponse(w, http.StatusOK, json)

	// var blocksa []*model.Block
	// var newBlock *model.Block
	// newBlock = &model.Block{}

	// blocksa = append(blocksa, newBlock)

	// hasComments := false
	// hasContents := false
	// for _, block := range blocksa {
	// 	block.BoardID = boardID
	// 	block.Type = "card"
	// 	block.Fields["properties"] = map[string]interface{}{"opportunityId": "Neue ID"}
	// }

	// if hasContents {
	// 	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionManageBoardCards) {
	// 		a.errorResponse(w, r, model.NewErrPermission("access denied to make board changes"))
	// 		return
	// 	}
	// }
	// if hasComments {
	// 	if !a.permissions.HasPermissionToBoard(userID, boardID, model.PermissionCommentBoardCards) {
	// 		a.errorResponse(w, r, model.NewErrPermission("access denied to post card comments"))
	// 		return
	// 	}
	// }

	// blocksa = model.GenerateBlockIDs(blocksa, a.logger)

	// // a.app.PatchBlock()

	// newBlocks, err := a.app.InsertBlocksAndNotify(blocks, userID, true)
	// if err != nil {
	// 	a.errorResponse(w, r, err)
	// 	return
	// }

	// _ = newBlocks

	// data, err := json.Marshal(abc)
	// if err != nil {
	// 	a.errorResponse(w, r, err)
	// 	return
	// }

	// jsonBytesResponse(w, http.StatusOK, data)
	// auditRec.Success()
}

func (a *API) handleArchiveImport(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /teams/{teamID}/archive/import archiveImport
	//
	// Import an archive of boards.
	//
	// ---
	// produces:
	// - application/json
	// consumes:
	// - multipart/form-data
	// parameters:
	// - name: teamID
	//   in: path
	//   description: Team ID
	//   required: true
	//   type: string
	// - name: file
	//   in: formData
	//   description: archive file to import
	//   required: true
	//   type: file
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	ctx := r.Context()
	session, _ := ctx.Value(sessionContextKey).(*model.Session)
	userID := session.UserID

	vars := mux.Vars(r)
	teamID := vars["teamID"]

	if !a.permissions.HasPermissionToTeam(userID, teamID, model.PermissionViewTeam) {
		a.errorResponse(w, r, model.NewErrPermission("access denied to create board"))
		return
	}

	isGuest, err := a.userIsGuest(userID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	if isGuest {
		a.errorResponse(w, r, model.NewErrPermission("access denied to create board"))
		return
	}

	file, handle, err := r.FormFile(UploadFormFileKey)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	defer file.Close()

	auditRec := a.makeAuditRecord(r, "import", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("filename", handle.Filename)
	auditRec.AddMeta("size", handle.Size)

	opt := model.ImportArchiveOptions{
		TeamID:     teamID,
		ModifiedBy: userID,
	}

	if err := a.app.ImportArchive(file, opt); err != nil {
		a.logger.Debug("Error importing archive",
			mlog.String("team_id", teamID),
			mlog.Err(err),
		)
		a.errorResponse(w, r, err)
		return
	}

	jsonStringResponse(w, http.StatusOK, "{}")
	auditRec.Success()
}

func (a *API) handleArchiveExportTeam(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /teams/{teamID}/archive/export archiveExportTeam
	//
	// Exports an archive of all blocks for all the boards in a team.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: teamID
	//   in: path
	//   description: Id of team
	//   required: true
	//   type: string
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     content:
	//       application-octet-stream:
	//         type: string
	//         format: binary
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	if a.MattermostAuth {
		a.errorResponse(w, r, model.NewErrNotImplemented("not permitted in plugin mode"))
		return
	}

	vars := mux.Vars(r)
	teamID := vars["teamID"]

	ctx := r.Context()
	session, _ := ctx.Value(sessionContextKey).(*model.Session)
	userID := session.UserID

	auditRec := a.makeAuditRecord(r, "archiveExportTeam", audit.Fail)

	defer a.audit.LogRecord(audit.LevelRead, auditRec)
	auditRec.AddMeta("TeamID", teamID)

	isGuest, err := a.userIsGuest(userID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	boards, err := a.app.GetBoardsForUserAndTeam(userID, teamID, !isGuest)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	ids := []string{}
	for _, board := range boards {
		ids = append(ids, board.ID)
	}

	opts := model.ExportArchiveOptions{
		TeamID:   teamID,
		BoardIDs: ids,
	}

	filename := fmt.Sprintf("archive-%s%s", time.Now().Format("2006-01-02"), archiveExtension)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Transfer-Encoding", "binary")

	if err := a.app.ExportArchive(w, opts); err != nil {
		a.errorResponse(w, r, err)
	}

	auditRec.Success()
}
