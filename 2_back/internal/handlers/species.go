package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"rkw_hatcher/internal/models"
)

type API struct {
	DB *sql.DB
}

func (a *API) Register(r *gin.Engine) {
	r.Use(cors())
	api := r.Group("/api")
	{
		api.GET("/health", a.health)

		api.GET("/egg-groups", a.listEggGroups)
		api.GET("/natures", a.listNatures)
		api.GET("/medals", a.listMedals)
		api.GET("/medal-types", a.listMedalTypes)

		api.GET("/species", a.listSpecies)
		api.POST("/species/sync/preview", a.syncSpeciesPreview)
		api.POST("/species/sync/commit", a.syncSpeciesCommit)
		api.GET("/species/:id/detail", a.getSpeciesDetail)
		api.GET("/species/:id/delete-impact", a.speciesDeleteImpact)
		api.GET("/species/:id", a.getSpecies)
		api.POST("/species", a.createSpecies)
		api.PUT("/species/:id", a.updateSpecies)
		api.DELETE("/species/:id", a.deleteSpecies)

		api.GET("/pets", a.listPets)
		api.GET("/pets/:id/delete-impact", a.petDeleteImpact)
		api.GET("/pets/:id", a.getPet)
		api.POST("/pets", a.createPet)
		api.PUT("/pets/:id", a.updatePet)
		api.DELETE("/pets/:id", a.deletePet)

		api.GET("/breeding-lines", a.listBreedingLines)
		api.POST("/breeding-lines", a.createBreedingLine)
		api.PUT("/breeding-lines/:id", a.updateBreedingLine)
		api.DELETE("/breeding-lines/:id", a.deleteBreedingLine)
		api.POST("/breeding-lines/:id/runs", a.startBreedingLineRun)
		api.DELETE("/breeding-lines/:id/runs/:runId", a.stopBreedingLineRun)

		api.GET("/breed/query", a.breedQuery)

		api.GET("/export/species", a.exportSpecies)
		api.POST("/import/species", a.importSpecies)
		api.GET("/export/pets", a.exportPets)
		api.POST("/import/pets", a.importPets)
	}
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 id"})
		return 0, false
	}
	return id, true
}

func (a *API) health(c *gin.Context) {
	if err := a.DB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "db": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "db": true})
}

func (a *API) listEggGroups(c *gin.Context) {
	rows, err := a.DB.Query(`SELECT id, name FROM egg_groups ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []models.EggGroup
	for rows.Next() {
		var g models.EggGroup
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, g)
	}
	c.JSON(http.StatusOK, list)
}

func (a *API) listNatures(c *gin.Context) {
	rows, err := a.DB.Query(`SELECT id, name, boost, penalty FROM natures ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []models.Nature
	for rows.Next() {
		var n models.Nature
		if err := rows.Scan(&n.ID, &n.Name, &n.Boost, &n.Penalty); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, n)
	}
	c.JSON(http.StatusOK, list)
}

func (a *API) listMedals(c *gin.Context) {
	rows, err := a.DB.Query(`SELECT id, type, name FROM medals ORDER BY type, id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []models.Medal
	for rows.Next() {
		var m models.Medal
		if err := rows.Scan(&m.ID, &m.Type, &m.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, m)
	}
	c.JSON(http.StatusOK, list)
}

func (a *API) listMedalTypes(c *gin.Context) {
	rows, err := a.DB.Query(`SELECT DISTINCT type FROM medals ORDER BY type`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, t)
	}
	c.JSON(http.StatusOK, list)
}

func nullStr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func nullUint64(v *uint64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func scanNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

func scanNullUint64(ns sql.NullInt64) *uint64 {
	if !ns.Valid {
		return nil
	}
	v := uint64(ns.Int64)
	return &v
}

func joinJSONArray(arr []string) ([]byte, error) {
	if arr == nil {
		arr = []string{}
	}
	return json.Marshal(arr)
}

func parseJSONArray(b []byte) ([]string, error) {
	var arr []string
	if len(b) == 0 {
		return []string{}, nil
	}
	err := json.Unmarshal(b, &arr)
	return arr, err
}

func validateEvoChain(name string, chain []string) string {
	if len(chain) == 0 {
		return "进化链不能为空"
	}
	if chain[len(chain)-1] != name {
		return "进化链末项必须等于最终形态名称"
	}
	return ""
}

func (a *API) loadSpeciesEggGroups(speciesID uint64) ([]uint64, []string, error) {
	rows, err := a.DB.Query(`
		SELECT e.id, e.name FROM species_egg_groups seg
		JOIN egg_groups e ON e.id = seg.egg_group_id
		WHERE seg.species_id = ? ORDER BY e.id`, speciesID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var ids []uint64
	var names []string
	for rows.Next() {
		var id uint64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		names = append(names, name)
	}
	return ids, names, nil
}

func (a *API) loadSpeciesBestNatures(speciesID uint64) ([]uint64, []string, error) {
	rows, err := a.DB.Query(`
		SELECT n.id, n.name FROM species_best_natures sbn
		JOIN natures n ON n.id = sbn.nature_id
		WHERE sbn.species_id = ? ORDER BY n.id`, speciesID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var ids []uint64
	var names []string
	for rows.Next() {
		var id uint64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		names = append(names, name)
	}
	return ids, names, nil
}

func (a *API) replaceSpeciesEggGroups(tx *sql.Tx, speciesID uint64, eggIDs []uint64) error {
	if _, err := tx.Exec(`DELETE FROM species_egg_groups WHERE species_id = ?`, speciesID); err != nil {
		return err
	}
	for _, eg := range eggIDs {
		if _, err := tx.Exec(`INSERT INTO species_egg_groups (species_id, egg_group_id) VALUES (?, ?)`, speciesID, eg); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) replaceSpeciesBestNatures(tx *sql.Tx, speciesID uint64, natureIDs []uint64) error {
	if _, err := tx.Exec(`DELETE FROM species_best_natures WHERE species_id = ?`, speciesID); err != nil {
		return err
	}
	for _, nid := range natureIDs {
		if _, err := tx.Exec(`INSERT INTO species_best_natures (species_id, nature_id) VALUES (?, ?)`, speciesID, nid); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) listSpecies(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	eggMode := strings.TrimSpace(c.Query("eggMode"))
	if eggMode != "intersect" {
		eggMode = "union"
	}
	var eggIDs []uint64
	if raw := strings.TrimSpace(c.Query("eggGroupIds")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseUint(p, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效蛋组 id"})
				return
			}
			eggIDs = append(eggIDs, id)
		}
	}

	sqlStr := `SELECT s.id, s."no", s.name, s.icon_url, s.evo_chain, s.notes, s.created_at, s.updated_at FROM species s`
	args := []interface{}{}
	where := []string{}

	if len(eggIDs) > 0 {
		ph := strings.Repeat("?,", len(eggIDs))
		ph = ph[:len(ph)-1]
		if eggMode == "intersect" {
			sqlStr += ` JOIN (
				SELECT species_id FROM species_egg_groups
				WHERE egg_group_id IN (` + ph + `)
				GROUP BY species_id
				HAVING COUNT(DISTINCT egg_group_id) = ?
			) egf ON egf.species_id = s.id`
			for _, id := range eggIDs {
				args = append(args, id)
			}
			args = append(args, len(eggIDs))
		} else {
			sqlStr += ` JOIN (
				SELECT DISTINCT species_id FROM species_egg_groups
				WHERE egg_group_id IN (` + ph + `)
			) egf ON egf.species_id = s.id`
			for _, id := range eggIDs {
				args = append(args, id)
			}
		}
	}

	if q != "" {
		like := "%" + q + "%"
		cond := `(s.name LIKE ? OR CAST(s.evo_chain AS TEXT) LIKE ? OR CAST(s."no" AS TEXT) LIKE ?)`
		args = append(args, like, like, like)
		if n, err := strconv.ParseUint(strings.TrimLeft(q, "0"), 10, 64); err == nil && n > 0 {
			cond = `(s.name LIKE ? OR CAST(s.evo_chain AS TEXT) LIKE ? OR CAST(s."no" AS TEXT) LIKE ? OR s."no" = ?)`
			args = append(args, n)
		}
		where = append(where, cond)
	}

	if len(where) > 0 {
		sqlStr += ` WHERE ` + strings.Join(where, " AND ")
	}
	sqlStr += ` ORDER BY s."no" IS NULL, s."no", s.id`

	rows, err := a.DB.Query(sqlStr, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []models.Species
	for rows.Next() {
		var s models.Species
		var evo []byte
		var no sql.NullInt64
		var icon, notes sql.NullString
		if err := rows.Scan(&s.ID, &no, &s.Name, &icon, &evo, &notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.No = scanNullUint64(no)
		s.EvoChain, _ = parseJSONArray(evo)
		s.IconURL = scanNullString(icon)
		s.Notes = scanNullString(notes)
		s.EggGroupIDs, s.EggGroupNames, _ = a.loadSpeciesEggGroups(s.ID)
		s.BestPvpNatureIDs, s.BestPvpNatureNames, _ = a.loadSpeciesBestNatures(s.ID)
		list = append(list, s)
	}
	if list == nil {
		list = []models.Species{}
	}
	c.JSON(http.StatusOK, list)
}

func (a *API) getSpecies(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	s, err := a.fetchSpecies(id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (a *API) fetchSpecies(id uint64) (models.Species, error) {
	var s models.Species
	var evo []byte
	var no sql.NullInt64
	var icon, notes sql.NullString
	err := a.DB.QueryRow(`
		SELECT s.id, s."no", s.name, s.icon_url, s.evo_chain, s.notes, s.created_at, s.updated_at
		FROM species s WHERE s.id = ?`, id).
		Scan(&s.ID, &no, &s.Name, &icon, &evo, &notes, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, err
	}
	s.No = scanNullUint64(no)
	s.EvoChain, _ = parseJSONArray(evo)
	s.IconURL = scanNullString(icon)
	s.Notes = scanNullString(notes)
	s.EggGroupIDs, s.EggGroupNames, err = a.loadSpeciesEggGroups(s.ID)
	if err != nil {
		return s, err
	}
	s.BestPvpNatureIDs, s.BestPvpNatureNames, err = a.loadSpeciesBestNatures(s.ID)
	return s, err
}

func (a *API) createSpecies(c *gin.Context) {
	var in models.SpeciesInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if msg := validateEvoChain(in.Name, in.EvoChain); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if len(in.EggGroupIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少勾选一个蛋组"})
		return
	}
	if len(in.BestPvpNatureIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少勾选一个最佳性格"})
		return
	}
	evo, _ := joinJSONArray(in.EvoChain)
	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO species ("no", name, icon_url, evo_chain, notes) VALUES (?, ?, ?, ?, ?)`,
		nullUint64(in.No), in.Name, nullStr(in.IconURL), evo, nullStr(in.Notes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id64, _ := res.LastInsertId()
	if err := a.replaceSpeciesEggGroups(tx, uint64(id64), in.EggGroupIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := a.replaceSpeciesBestNatures(tx, uint64(id64), in.BestPvpNatureIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s, _ := a.fetchSpecies(uint64(id64))
	c.JSON(http.StatusCreated, s)
}

func (a *API) updateSpecies(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in models.SpeciesInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(in.BestPvpNatureIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少勾选一个最佳性格"})
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var exists uint64
	if err := tx.QueryRow(`SELECT id FROM species WHERE id=?`, id).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 仅允许修改推荐性格，避免误清空图鉴编号等字段
	if err := a.replaceSpeciesBestNatures(tx, id, in.BestPvpNatureIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(`UPDATE species SET updated_at=CURRENT_TIMESTAMP WHERE id=?`, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s, _ := a.fetchSpecies(id)
	c.JSON(http.StatusOK, s)
}

func (a *API) speciesPetIDs(speciesID uint64) ([]uint64, error) {
	rows, err := a.DB.Query(`SELECT id FROM my_pets WHERE species_id=?`, speciesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (a *API) speciesDeleteImpact(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var name string
	if err := a.DB.QueryRow(`SELECT name FROM species WHERE id=?`, id).Scan(&name); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	pets, err := a.speciesPetIDs(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, err := a.DB.Query(`SELECT name FROM breeding_lines WHERE target_species_id=? ORDER BY id`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		names = append(names, n)
	}
	c.JSON(http.StatusOK, models.SpeciesDeleteImpact{
		Name:      name,
		PetCount:  len(pets),
		LineCount: len(names),
		LineNames: names,
	})
}

func (a *API) deleteSpecies(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var exists uint64
	if err := a.DB.QueryRow(`SELECT id FROM species WHERE id=?`, id).Scan(&exists); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	petIDs, err := a.speciesPetIDs(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	petSet := map[uint64]struct{}{}
	for _, pid := range petIDs {
		petSet[pid] = struct{}{}
	}

	lineRows, err := a.DB.Query(`SELECT id FROM breeding_lines WHERE target_species_id=?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var targetLines []uint64
	targetSet := map[uint64]struct{}{}
	for lineRows.Next() {
		var lid uint64
		if err := lineRows.Scan(&lid); err != nil {
			lineRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		targetLines = append(targetLines, lid)
		targetSet[lid] = struct{}{}
	}
	lineRows.Close()

	otherSet := map[uint64]struct{}{}
	for _, pid := range petIDs {
		rows, err := a.DB.Query(`SELECT DISTINCT line_id FROM breeding_line_scheme_pets WHERE pet_id=?`, pid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for rows.Next() {
			var lid uint64
			if err := rows.Scan(&lid); err != nil {
				rows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if _, skip := targetSet[lid]; !skip {
				otherSet[lid] = struct{}{}
			}
		}
		rows.Close()
	}

	type linePlan struct {
		id      uint64
		schemes []models.BreedingScheme
	}
	var plans []linePlan
	for lid := range otherSet {
		var speciesID, natureID uint64
		if err := a.DB.QueryRow(`SELECT target_species_id, expected_nature_id FROM breeding_lines WHERE id=?`, lid).
			Scan(&speciesID, &natureID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		medals, err := a.loadLineMedals(lid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		schemes, err := a.buildBreedSchemes(speciesID, natureID, medals, lid, petIDs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		plans = append(plans, linePlan{id: lid, schemes: schemes})
	}

	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var partners []uint64
	for _, lid := range targetLines {
		runRows, err := tx.Query(`SELECT stud_pet_id, dam_pet_id FROM breeding_line_runs WHERE line_id=?`, lid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for runRows.Next() {
			var stud, dam uint64
			if err := runRows.Scan(&stud, &dam); err != nil {
				runRows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			partners = append(partners, stud, dam)
		}
		runRows.Close()
		if _, err := tx.Exec(`DELETE FROM breeding_lines WHERE id=?`, lid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	for _, pid := range petIDs {
		runRows, err := tx.Query(`SELECT id, stud_pet_id, dam_pet_id FROM breeding_line_runs WHERE stud_pet_id=? OR dam_pet_id=?`, pid, pid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		type runRow struct{ id, stud, dam uint64 }
		var runs []runRow
		for runRows.Next() {
			var r runRow
			if err := runRows.Scan(&r.id, &r.stud, &r.dam); err != nil {
				runRows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			runs = append(runs, r)
		}
		runRows.Close()
		for _, r := range runs {
			if _, err := tx.Exec(`DELETE FROM breeding_line_runs WHERE id=?`, r.id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			partners = append(partners, r.stud, r.dam)
		}
	}

	if _, err := tx.Exec(`DELETE FROM my_pets WHERE species_id=?`, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	release := make([]uint64, 0, len(partners))
	for _, pid := range partners {
		if _, gone := petSet[pid]; !gone {
			release = append(release, pid)
		}
	}
	if err := a.releasePetsIfNotInRuns(tx, release); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, plan := range plans {
		if len(plan.schemes) == 0 {
			if _, err := tx.Exec(`DELETE FROM breeding_lines WHERE id=?`, plan.id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			continue
		}
		if err := a.replaceLineSchemes(tx, plan.id, plan.schemes); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := a.dropInvalidLineRuns(tx, plan.id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	res, err := tx.Exec(`DELETE FROM species WHERE id=?`, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
