package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"rkw_hatcher/internal/models"
)

func (a *API) listBreedingLines(c *gin.Context) {
	rows, err := a.DB.Query(`
		SELECT b.id, b.name, b.target_species_id, s.name, s.icon_url, b.expected_nature_id, n.name,
		       b.status, b.steps_note, b.created_at, b.updated_at
		FROM breeding_lines b
		JOIN species s ON s.id = b.target_species_id
		JOIN natures n ON n.id = b.expected_nature_id
		ORDER BY b.id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []models.BreedingLine
	for rows.Next() {
		var b models.BreedingLine
		var note sql.NullString
		var icon sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &b.TargetSpeciesID, &b.TargetSpeciesName, &icon,
			&b.ExpectedNatureID, &b.ExpectedNatureName, &b.Status, &note, &b.CreatedAt, &b.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if icon.Valid {
			b.TargetSpeciesIcon = icon.String
		}
		b.StepsNote = scanNullString(note)
		b.Medals, _ = a.loadLineMedals(b.ID)
		b.Schemes, _ = a.loadLineSchemes(b.ID)
		b.MaxSlots = calcLineMaxSlots(b.Schemes)
		b.Runs, _ = a.loadLineRuns(b.ID)
		list = append(list, b)
	}
	if list == nil {
		list = []models.BreedingLine{}
	}
	c.JSON(http.StatusOK, list)
}

func (a *API) normalizeLineInput(in *models.BreedingLineInput) error {
	if in.TargetSpeciesID == 0 || in.ExpectedNatureID == 0 {
		return simpleError("目标图鉴与性格必填")
	}
	if in.Status == "" {
		in.Status = "进行中"
	}
	if strings.TrimSpace(in.Name) == "" {
		var spName, natName string
		_ = a.DB.QueryRow(`SELECT name FROM species WHERE id=?`, in.TargetSpeciesID).Scan(&spName)
		_ = a.DB.QueryRow(`SELECT name FROM natures WHERE id=?`, in.ExpectedNatureID).Scan(&natName)
		in.Name = strings.TrimSpace(spName + "·" + natName)
		if in.Name == "·" || in.Name == "" {
			in.Name = "未命名产线"
		}
	}
	if in.Medals == nil {
		in.Medals = []models.PetMedalPick{}
	}
	return nil
}

func (a *API) createBreedingLine(c *gin.Context) {
	var in models.BreedingLineInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := a.normalizeLineInput(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conflict, err := a.lineMedalsConflict(in.TargetSpeciesID, in.ExpectedNatureID, in.Medals, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if conflict {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已存在相同目标精灵、性格与奖章的产线"})
		return
	}
	schemes, err := a.buildBreedSchemes(in.TargetSpeciesID, in.ExpectedNatureID, in.Medals, 0, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(schemes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "暂无推荐方案，无法保存"})
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO breeding_lines (name, target_species_id, expected_nature_id, status, steps_note)
		VALUES (?,?,?,?,?)`,
		in.Name, in.TargetSpeciesID, in.ExpectedNatureID, in.Status, nullStr(in.StepsNote))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id64, _ := res.LastInsertId()
	lineID := uint64(id64)
	if err := a.replaceLineMedals(tx, lineID, in.Medals); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := a.replaceLineSchemes(tx, lineID, schemes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	b, _ := a.fetchBreedingLine(lineID)
	c.JSON(http.StatusCreated, b)
}

func (a *API) updateBreedingLine(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in models.BreedingLineInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := a.normalizeLineInput(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conflict, err := a.lineMedalsConflict(in.TargetSpeciesID, in.ExpectedNatureID, in.Medals, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if conflict {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已存在相同目标精灵、性格与奖章的产线"})
		return
	}
	schemes, err := a.buildBreedSchemes(in.TargetSpeciesID, in.ExpectedNatureID, in.Medals, id, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(schemes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "暂无推荐方案，无法保存"})
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE breeding_lines SET name=?, target_species_id=?, expected_nature_id=?, status=?, steps_note=?
		WHERE id=?`,
		in.Name, in.TargetSpeciesID, in.ExpectedNatureID, in.Status, nullStr(in.StepsNote), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists uint64
		if err := tx.QueryRow(`SELECT id FROM breeding_lines WHERE id=?`, id).Scan(&exists); err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := a.replaceLineMedals(tx, id, in.Medals); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := a.replaceLineSchemes(tx, id, schemes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := a.dropInvalidLineRuns(tx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	b, _ := a.fetchBreedingLine(id)
	c.JSON(http.StatusOK, b)
}

func (a *API) deleteBreedingLine(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	runRows, err := tx.Query(`SELECT stud_pet_id, dam_pet_id FROM breeding_line_runs WHERE line_id=?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var petIDs []uint64
	for runRows.Next() {
		var stud, dam uint64
		if err := runRows.Scan(&stud, &dam); err != nil {
			runRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		petIDs = append(petIDs, stud, dam)
	}
	runRows.Close()

	res, err := tx.Exec(`DELETE FROM breeding_lines WHERE id=?`, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
		return
	}
	if err := a.releasePetsIfNotInRuns(tx, petIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) startBreedingLineRun(c *gin.Context) {
	lineID, ok := parseID(c)
	if !ok {
		return
	}
	var in models.BreedingLineRunInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.StudPetID == 0 || in.DamPetID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择一公一母"})
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var exists uint64
	if err := tx.QueryRow(`SELECT id FROM breeding_lines WHERE id=?`, lineID).Scan(&exists); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "产线不存在"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	okStud, err := a.petInLineScheme(tx, lineID, in.StudPetID, "stud")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	okDam, err := a.petInLineScheme(tx, lineID, in.DamPetID, "dam")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !okStud || !okDam {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所选精灵不在该产线方案内"})
		return
	}

	var studStatus, damStatus, studGender, damGender string
	if err := tx.QueryRow(`SELECT gender, status FROM my_pets WHERE id=?`, in.StudPetID).Scan(&studGender, &studStatus); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "种公不存在"})
		return
	}
	if err := tx.QueryRow(`SELECT gender, status FROM my_pets WHERE id=?`, in.DamPetID).Scan(&damGender, &damStatus); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "种母不存在"})
		return
	}
	if studGender != "公" || damGender != "母" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需选择一公一母"})
		return
	}
	if studStatus != "空闲中" || damStatus != "空闲中" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所选精灵忙碌中，无法加入生产"})
		return
	}

	schemes, err := a.loadLineSchemes(lineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	maxSlots := calcLineMaxSlots(schemes)
	lineCount, err := a.countLineRuns(tx, lineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if lineCount >= maxSlots {
		c.JSON(http.StatusBadRequest, gin.H{"error": "本产线生产槽已满"})
		return
	}
	globalCount, err := a.countGlobalRuns(tx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if globalCount >= 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "全局生产槽已满(5/5)"})
		return
	}

	res, err := tx.Exec(`
		INSERT INTO breeding_line_runs (line_id, stud_pet_id, dam_pet_id) VALUES (?,?,?)`,
		lineID, in.StudPetID, in.DamPetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := a.setPetsStatus(tx, []uint64{in.StudPetID, in.DamPetID}, "忙碌中"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(`UPDATE breeding_lines SET status='进行中' WHERE id=?`, lineID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = res
	b, _ := a.fetchBreedingLine(lineID)
	c.JSON(http.StatusCreated, b)
}

func (a *API) stopBreedingLineRun(c *gin.Context) {
	lineID, ok := parseID(c)
	if !ok {
		return
	}
	runID, err := strconv.ParseUint(c.Param("runId"), 10, 64)
	if err != nil || runID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 runId"})
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var studID, damID, runLine uint64
	err = tx.QueryRow(`SELECT line_id, stud_pet_id, dam_pet_id FROM breeding_line_runs WHERE id=?`, runID).
		Scan(&runLine, &studID, &damID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "生产队不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if runLine != lineID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生产队不属于该产线"})
		return
	}

	if _, err := tx.Exec(`DELETE FROM breeding_line_runs WHERE id=?`, runID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := a.releasePetsIfNotInRuns(tx, []uint64{studID, damID}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	b, _ := a.fetchBreedingLine(lineID)
	c.JSON(http.StatusOK, b)
}

func nullUint(v *uint64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// GET /api/breed/query?speciesId=&gender=公|母&natureId=
// 公：查同蛋组且最佳性格含 natureId 的目标图鉴（按母用途），附带背包母宠
// 母：查同蛋组且性格属于所选精灵最佳性格的公；表单 natureId 必传但不参与公侧筛选；附带同蛋组图鉴
func (a *API) breedQuery(c *gin.Context) {
	speciesID, err := strconv.ParseUint(strings.TrimSpace(c.Query("speciesId")), 10, 64)
	if err != nil || speciesID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择精灵"})
		return
	}
	gender := strings.TrimSpace(c.Query("gender"))
	if gender != "公" && gender != "母" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择性别"})
		return
	}
	natureID, err := strconv.ParseUint(strings.TrimSpace(c.Query("natureId")), 10, 64)
	if err != nil || natureID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择性格"})
		return
	}

	var (
		mode    string
		results []models.BreedQueryResult
	)

	if gender == "公" {
		mode = "stud"
		sqlStr := `
			SELECT DISTINCT s.id, s.` + "`no`" + `, s.name, s.icon_url, s.evo_chain, s.notes, s.created_at, s.updated_at
			FROM species s
			JOIN species_best_natures sbn ON sbn.species_id = s.id AND sbn.nature_id = ?
			WHERE EXISTS (
				SELECT 1 FROM species_egg_groups a
				JOIN species_egg_groups b ON a.egg_group_id = b.egg_group_id
				JOIN egg_groups e ON e.id = a.egg_group_id
				WHERE a.species_id = s.id AND b.species_id = ?
				  AND e.name <> '无法孵蛋'
			)
			ORDER BY s.` + "`no`" + ` IS NULL, s.` + "`no`" + `, s.id`
		rows, err := a.DB.Query(sqlStr, natureID, speciesID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		for rows.Next() {
			s, err := scanBreedQuerySpecies(rows)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			s.EggGroupIDs, s.EggGroupNames, _ = a.loadSpeciesEggGroups(s.ID)
			s.BestPvpNatureIDs, s.BestPvpNatureNames, _ = a.loadSpeciesBestNatures(s.ID)
			pets, _ := a.loadPetsBySpeciesGender(s.ID, "母")
			results = append(results, models.BreedQueryResult{Species: s, Pets: pets})
		}
	} else {
		mode = "dam"
		bestIDs, _, err := a.loadSpeciesBestNatures(speciesID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if len(bestIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"mode":      mode,
				"speciesId": speciesID,
				"natureId":  natureID,
				"results":   []models.BreedQueryResult{},
			})
			return
		}

		sqlStr := `
			SELECT DISTINCT s.id, s.` + "`no`" + `, s.name, s.icon_url, s.evo_chain, s.notes, s.created_at, s.updated_at
			FROM species s
			WHERE EXISTS (
				SELECT 1 FROM species_egg_groups a
				JOIN species_egg_groups b ON a.egg_group_id = b.egg_group_id
				JOIN egg_groups e ON e.id = a.egg_group_id
				WHERE a.species_id = s.id AND b.species_id = ?
				  AND e.name <> '无法孵蛋'
			)
			ORDER BY (s.id = ?) DESC, s.` + "`no`" + ` IS NULL, s.` + "`no`" + `, s.id`
		rows, err := a.DB.Query(sqlStr, speciesID, speciesID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		bestSet := make(map[uint64]struct{}, len(bestIDs))
		damNatureOK := false
		for _, id := range bestIDs {
			bestSet[id] = struct{}{}
			if id == natureID {
				damNatureOK = true
			}
		}
		for rows.Next() {
			s, err := scanBreedQuerySpecies(rows)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			s.EggGroupIDs, s.EggGroupNames, _ = a.loadSpeciesEggGroups(s.ID)
			s.BestPvpNatureIDs, s.BestPvpNatureNames, _ = a.loadSpeciesBestNatures(s.ID)
			males, _ := a.loadPetsBySpeciesGender(s.ID, "公")
			pets := make([]models.MyPet, 0, len(males))
			for _, p := range males {
				if damNatureOK {
					// 母已是正确性格：只推同性格公，目标产下该性格子代
					if p.NatureID == natureID {
						pets = append(pets, p)
					}
				} else if _, ok := bestSet[p.NatureID]; ok {
					pets = append(pets, p)
				}
			}
			// 无匹配公时：仅保留母本物种图鉴行，其它空物种不展示
			if len(pets) == 0 && s.ID != speciesID {
				continue
			}
			results = append(results, models.BreedQueryResult{Species: s, Pets: pets})
		}
	}

	if results == nil {
		results = []models.BreedQueryResult{}
	}
	c.JSON(http.StatusOK, gin.H{
		"mode":      mode,
		"speciesId": speciesID,
		"natureId":  natureID,
		"results":   results,
	})
}

type speciesRowScanner interface {
	Scan(dest ...interface{}) error
}

func scanBreedQuerySpecies(rows speciesRowScanner) (models.Species, error) {
	var s models.Species
	var evo []byte
	var no sql.NullInt64
	var icon, notes sql.NullString
	if err := rows.Scan(&s.ID, &no, &s.Name, &icon, &evo, &notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return s, err
	}
	s.No = scanNullUint64(no)
	s.EvoChain, _ = parseJSONArray(evo)
	s.IconURL = scanNullString(icon)
	s.Notes = scanNullString(notes)
	return s, nil
}

func (a *API) loadPetsBySpeciesGender(speciesID uint64, gender string) ([]models.MyPet, error) {
	prows, err := a.DB.Query(`
		SELECT p.id, p.species_id, s2.name, p.gender, p.nature_id, n2.name, p.status, p.created_at, p.updated_at
		FROM my_pets p
		JOIN species s2 ON s2.id = p.species_id
		JOIN natures n2 ON n2.id = p.nature_id
		WHERE p.species_id = ? AND p.gender = ?
		ORDER BY p.id`, speciesID, gender)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	pets := []models.MyPet{}
	for prows.Next() {
		var p models.MyPet
		if err := prows.Scan(&p.ID, &p.SpeciesID, &p.SpeciesName, &p.Gender, &p.NatureID, &p.NatureName, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return pets, err
		}
		p.Medals, _ = a.loadPetMedals(p.ID)
		pets = append(pets, p)
	}
	return pets, nil
}
