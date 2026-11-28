package view

import (
	"strings"
	"testing"

	"ragnarok-of-building/internal/calc"
	"ragnarok-of-building/internal/gamedata"
)

// TestGoodAndEvil: every class has its crown, boots and set weapons; the
// enchants fit, every bonus of their scripts is understood, and the sets
// change the numbers.
func TestGoodAndEvil(t *testing.T) {
	c, v := stressCatalog(t)
	slotOf := func(it *gamedata.Item) string {
		for _, sl := range calc.Slots {
			if calc.Fits(it, sl) {
				return sl
			}
		}
		return ""
	}
	preview, crowns, boots, weapons := 0, map[int]int{}, map[int]int{}, map[int]int{}
	for id, it := range c.Items {
		// The gear, not the refining hammers that share its name.
		ge := strings.Contains(it.Name["en"], "of Good and Evil (") && (it.Type == "Armor" || it.Type == "Weapon")
		if it.Preview {
			preview++
			if !strings.Contains(it.Name["pt"], "(em breve)") || len(it.Desc["pt"]) == 0 {
				t.Errorf("%d %v: name or description missing", id, it.Name)
			}
			if it.SubType != "Enchant" && (slotOf(it) == "" || it.Script == "") {
				t.Errorf("%d %s: no slot or bonuses", id, it.Name["pt"])
			}
		}
		if !ge && !(it.Preview && it.Type == "Weapon") {
			continue
		}
		who := 0
		for jid, j := range c.Jobs {
			if !j.Fourth || !j.CanEquip(it) {
				continue
			}
			who++
			switch {
			case it.Type == "Weapon":
				weapons[jid]++
			case calc.Fits(it, calc.Shoes):
				boots[jid]++
			default:
				crowns[jid]++
			}
		}
		if who == 0 {
			t.Errorf("%d %s: nobody can wear it", id, it.Name["pt"])
		}
		if len(c.Enchants[id][2]) == 0 || len(c.Enchants[id][3]) == 0 {
			t.Errorf("%d %s: no enchants", id, it.Name["pt"])
		}
	}
	for jid, j := range c.Jobs {
		if j.Fourth && len(j.Tree) > 0 && (crowns[jid] == 0 || boots[jid] != 1) {
			t.Errorf("%d %s: %d crowns, %d boots", jid, j.Code, crowns[jid], boots[jid])
		}
	}
	t.Logf("%d preview items, weapons per class %v", preview, weapons)

	// Dragon Knight: Execution Great Sword +9 [B], crown +9, Dom Infernal:
	// Hack and Slash +12 (4 x 9/3) +10 (B) +15 (Dom Infernal).
	b := calc.Build{Job: 4252, BaseLv: 250, JobLv: 50, Stats: [6]int{100, 100, 100, 100, 100, 100}, Skills: map[int]int{}, Equip: map[string]calc.Equip{
		calc.Weapon:  {Item: 600041, Refine: 9, Grade: 3},
		calc.HeadTop: {Item: 400374, Refine: 9, Cards: [4]int{0, 0, 312021}},
	}}
	calc.Normalize(&b, c)
	if b.Equip[calc.HeadTop].Item != 400374 || b.Equip[calc.HeadTop].Cards[2] != 312021 || b.Equip[calc.Weapon].Item != 600041 {
		t.Fatalf("not equipped: %+v", b.Equip)
	}
	with := calc.Compute(b, c)
	nb := cloneBuild(b)
	delete(nb.Equip, calc.HeadTop)
	without := calc.Compute(nb, c)
	diff := map[string]int{}
	for k, x := range with.Extra {
		if strings.Contains(k, "DK_HACKANDSLASHER") {
			diff[k] = x - without.Extra[k]
		}
	}
	found := false
	for _, x := range diff {
		found = found || x == 37
	}
	if !found {
		t.Errorf("Hack and Slash from the set: %v, want +37", diff)
	}
	_ = v
}

// TestDomSets: both Doms tell every class's set, once, and the sets LATAM
// has already (crown, weapon and Dom) count once.
func TestDomSets(t *testing.T) {
	c, _ := stressCatalog(t)
	if _, err := New(Options{Catalog: c}); err != nil { // again: no repeated text
		t.Fatal(err)
	}
	for _, dom := range []int{312014, 312021} {
		text := c.Items[dom].Desc["pt"]
		n, titles := 0, 0
		for _, ln := range text {
			if strings.Contains(ln, "Conjunto^") {
				n++
			}
			if strings.Contains(ln, "ainda não chegaram") {
				titles++
			}
		}
		if n != 23 || titles != 1 { // one per crown, two for Troubadour/Trouvere
			t.Errorf("%s: %d sets, %d titles, want 23 and 1", c.Items[dom].Name["pt"], n, titles)
		}
		full := strings.Join(text, "\n")
		for _, cb := range c.Combos {
			if len(cb.Items) == 3 && cb.Items[0] >= 400374 && cb.Items[0] <= 400395 {
				for _, id := range cb.Items {
					w := c.Items[id]
					has := false
					for _, x := range cb.Items {
						has = has || x == dom
					}
					if has && w.Type == "Weapon" && !strings.Contains(full, w.Name["pt"]) {
						t.Errorf("%s does not tell the set with %s", c.Items[dom].Name["pt"], w.Name["pt"])
					}
				}
			}
		}
	}
	for id, it := range c.Items {
		if it.Preview && it.SubType == "Enchant" && strings.HasPrefix(it.Name["pt"], "Essência") && !strings.Contains(strings.Join(it.Desc["pt"], "\n"), "Conjunto^") {
			t.Errorf("%d %s tells no set", id, it.Name["pt"])
		}
	}
	// Arcana crown + Báculo Celestial + Dom Celestial: Gelo Cortante +15 once.
	for dom, want := range map[int]int{0: 0, 312014: 15} {
		b := calc.Build{Job: 4255, BaseLv: 250, JobLv: 50, Stats: [6]int{1, 1, 1, 1, 1, 1}, Equip: map[string]calc.Equip{
			calc.Weapon: {Item: 640037}, calc.HeadTop: {Item: 400376, Cards: [4]int{0, 0, dom}}}}
		calc.Normalize(&b, c)
		r := calc.Compute(b, c)
		nb := cloneBuild(b)
		e := nb.Equip[calc.HeadTop]
		e.Cards[2] = 0
		nb.Equip[calc.HeadTop] = e
		r0 := calc.Compute(nb, c)
		for k, x := range r.Extra {
			if strings.HasPrefix(k, "bSkillAtk") && strings.Contains(k, "AG_FROZEN_SLASH") && x-r0.Extra[k] != want {
				t.Errorf("Dom %d: Gelo Cortante +%d, want +%d", dom, x-r0.Extra[k], want)
			}
		}
	}
}
