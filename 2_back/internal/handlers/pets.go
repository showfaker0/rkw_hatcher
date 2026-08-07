package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"rkw_hatcher/internal/models"
)

func (a *API) loadPetMedals(petID uint64) ([]models.PetMedalPick, error) {
	rows, err := a.DB.Query(`
		SELECT mpm.medal_type, mpm.medal_id, m.name
		FROM my_pet_medals mpm JOIN medals m ON m.id = mpm.medal_id
		WHERE mpm.my_pet_id = ? ORDER BY mpm.medal_type`, petID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.PetMedalPick
	for rows.Next() {
		var p models.PetMedalPick
		if err := rows.Scan(&p.MedalType, &p.MedalID, &p.MedalName); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	if list == nil {
		list = []models.PetMedalPick{}
	}
	return list, nil
}

func (a *API) replacePetMedals(tx *sql.Tx, petID uint64, picks []models.PetMedalPick) error {
	if _, err := tx.Exec(`DELETE FROM my_pet_medals WHERE my_pet_id = ?`, petID); err != nil {
		return err
	}
	for _, p := range picks {
		if p.MedalID == 0 || strings.TrimSpace(p.MedalType) == "" {
			continue
		}
		var typ string
		if err := tx.QueryRow(`SELECT type FROM medals WHERE id = ?`, p.MedalID).Scan(&typ); err != nil {
			return err
		}
		if typ != p.MedalType {
			return errMedalTypeMismatch
		}
		if _, err := tx.Exec(`INSERT INTO my_pet_medals (my_pet_id, medal_type, medal_id) VALUES (?, ?, ?)`,
			petID, p.MedalType, p.MedalID); err != nil {
			return err
		}
	}
	return nil
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

const errMedalTypeMismatch = simpleError("奖章与类型不匹配")

func (a *API) fetchPet(id uint64) (models.MyPet, error) {
	var p models.MyPet
	err := a.DB.QueryRow(`
		SELECT p.id, p.species_id, s.name, p.gender, p.nature_id, n.name, p.status, p.created_at, p.updated_at
		FROM my_pets p
		JOIN species s ON s.id = p.species_id
		JOIN natures n ON n.id = p.nature_id
		WHERE p.id = ?`, id).
		Scan(&p.ID, &p.SpeciesID, &p.SpeciesName, &p.Gender, &p.NatureID, &p.NatureName, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, err
	}
	p.Medals, err = a.loadPetMedals(p.ID)
	return p, err
}

func (a *API) listPets(c *gin.Context) {
	sqlStr := `
		SELECT p.id, p.species_id, s.name, p.gender, p.nature_id, n.name, p.status, p.created_at, p.updated_at
		FROM my_pets p
		JOIN species s ON s.id = p.species_id
		JOIN natures n ON n.id = p.nature_id
		WHERE 1=1`
	args := []interface{}{}
	if v := c.Query("speciesId"); v != "" {
		sqlStr += ` AND p.species_id = ?`
		args = append(args, v)
	}
	if v := c.Query("natureId"); v != "" {
		sqlStr += ` AND p.nature_id = ?`
		args = append(args, v)
	}
	if v := c.Query("gender"); v != "" {
		sqlStr += ` AND p.gender = ?`
		args = append(args, v)
	}
	if v := c.Query("status"); v != "" {
		sqlStr += ` AND p.status = ?`
		args = append(args, v)
	}
	sqlStr += ` ORDER BY p.id DESC`
	rows, err := a.DB.Query(sqlStr, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []models.MyPet
	for rows.Next() {
		var p models.MyPet
		if err := rows.Scan(&p.ID, &p.SpeciesID, &p.SpeciesName, &p.Gender, &p.NatureID, &p.NatureName, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		p.Medals, _ = a.loadPetMedals(p.ID)
		list = append(list, p)
	}
	if list == nil {
		list = []models.MyPet{}
	}
	c.JSON(http.StatusOK, list)
}

func (a *API) getPet(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := a.fetchPet(id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (a *API) createPet(c *gin.Context) {
	var in models.MyPetInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.Gender != "公" && in.Gender != "母" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "性别必须是公或母"})
		return
	}
	if in.SpeciesID == 0 || in.NatureID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图鉴与性格必填"})
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO my_pets (species_id, gender, nature_id) VALUES (?,?,?)`,
		in.SpeciesID, in.Gender, in.NatureID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id64, _ := res.LastInsertId()
	if err := a.replacePetMedals(tx, uint64(id64), in.Medals); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	p, _ := a.fetchPet(uint64(id64))
	c.JSON(http.StatusCreated, p)
}

func (a *API) updatePet(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in models.MyPetInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.Gender != "公" && in.Gender != "母" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "性别必须是公或母"})
		return
	}
	if in.SpeciesID == 0 || in.NatureID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图鉴与性格必填"})
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE my_pets SET species_id=?, gender=?, nature_id=? WHERE id=?`,
		in.SpeciesID, in.Gender, in.NatureID, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// 字段未变化时 RowsAffected 可能为 0，不能据此判定不存在
		var exists uint64
		if err := tx.QueryRow(`SELECT id FROM my_pets WHERE id=?`, id).Scan(&exists); err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := a.replacePetMedals(tx, id, in.Medals); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	p, _ := a.fetchPet(id)
	c.JSON(http.StatusOK, p)
}

func (a *API) deletePet(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var runN int
	if err := a.DB.QueryRow(`
		SELECT COUNT(*) FROM breeding_line_runs
		WHERE stud_pet_id=? OR dam_pet_id=?`, id, id).Scan(&runN); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if runN > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该精灵正在生产中，请先关闭对应产线生产后再删除"})
		return
	}

	lineRows, err := a.DB.Query(`
		SELECT DISTINCT b.id, b.target_species_id, b.expected_nature_id
		FROM breeding_line_scheme_pets blsp
		JOIN breeding_lines b ON b.id = blsp.line_id
		WHERE blsp.pet_id=?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type linePlan struct {
		id      uint64
		schemes []models.BreedingScheme
	}
	var plans []linePlan
	for lineRows.Next() {
		var lineID, speciesID, natureID uint64
		if err := lineRows.Scan(&lineID, &speciesID, &natureID); err != nil {
			lineRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		medals, err := a.loadLineMedals(lineID)
		if err != nil {
			lineRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		schemes, err := a.buildBreedSchemes(speciesID, natureID, medals, lineID, id)
		if err != nil {
			lineRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		plans = append(plans, linePlan{id: lineID, schemes: schemes})
	}
	lineRows.Close()

	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM my_pets WHERE id=?`, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
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

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) petDeleteImpact(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var exists uint64
	if err := a.DB.QueryRow(`SELECT id FROM my_pets WHERE id=?`, id).Scan(&exists); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := models.PetDeleteImpact{Lines: []models.PetDeleteLineImpact{}}

	var runN int
	if err := a.DB.QueryRow(`
		SELECT COUNT(*) FROM breeding_line_runs
		WHERE stud_pet_id=? OR dam_pet_id=?`, id, id).Scan(&runN); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if runN > 0 {
		out.Blocked = true
		out.InProduction = true
		out.Message = "该精灵正在生产中，请先关闭对应产线生产后再删除"
		// 仍返回涉及产线名称便于展示
		rows, err := a.DB.Query(`
			SELECT DISTINCT b.id, b.name, s.name, n.name
			FROM breeding_line_runs r
			JOIN breeding_lines b ON b.id = r.line_id
			JOIN species s ON s.id = b.target_species_id
			JOIN natures n ON n.id = b.expected_nature_id
			WHERE r.stud_pet_id=? OR r.dam_pet_id=?
			ORDER BY b.id DESC`, id, id)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var item models.PetDeleteLineImpact
				if err := rows.Scan(&item.LineID, &item.LineName, &item.TargetSpeciesName, &item.ExpectedNatureName); err != nil {
					break
				}
				out.Lines = append(out.Lines, item)
			}
		}
		c.JSON(http.StatusOK, out)
		return
	}

	rows, err := a.DB.Query(`
		SELECT DISTINCT b.id, b.name, s.name, n.name, b.target_species_id, b.expected_nature_id
		FROM breeding_line_scheme_pets blsp
		JOIN breeding_lines b ON b.id = blsp.line_id
		JOIN species s ON s.id = b.target_species_id
		JOIN natures n ON n.id = b.expected_nature_id
		WHERE blsp.pet_id=?
		ORDER BY b.id DESC`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.PetDeleteLineImpact
		var speciesID, natureID uint64
		if err := rows.Scan(&item.LineID, &item.LineName, &item.TargetSpeciesName, &item.ExpectedNatureName, &speciesID, &natureID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		medals, _ := a.loadLineMedals(item.LineID)
		schemes, err := a.buildBreedSchemes(speciesID, natureID, medals, item.LineID, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		item.WillRemove = len(schemes) == 0
		out.Lines = append(out.Lines, item)
	}
	c.JSON(http.StatusOK, out)
}
