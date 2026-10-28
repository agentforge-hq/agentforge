package agentforge

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func adoptionResult(t *testing.T, files map[string]string) selectedSkill {
	t.Helper()
	sha := strings.Repeat("a", 40)
	s := selectedSkill{SkillID: "owner/repo/skills/example", Name: "example", AdoptionStatus: "review_required",
		Source: CatalogSource{Repository: "owner/repo", Revision: sha, SkillPath: "skills/example", ManifestSHA256: hashBytes([]byte(files["SKILL.md"])), URL: "https://github.com/owner/repo/tree/" + sha + "/skills/example"}}
	for _, path := range sortedKeys(files) {
		s.Source.Files = append(s.Source.Files, SourceFile{Path: path, GitBlobSHA1: gitBlobSHA1([]byte(files[path]))})
	}
	return s
}

func resultJSON(t *testing.T, selected selectedSkill) []byte {
	t.Helper()
	data, err := json.Marshal(M{"rows": []selectedSkill{selected}, "meta": M{"status": "matches"}})
	must(t, err)
	return data
}

func mockPackage(t *testing.T, f *fixture, selected selectedSkill, files map[string]string) *[]string {
	t.Helper()
	entries := []zipEntry{}
	for _, path := range sortedKeys(files) {
		prefix := "repo/" + selected.Source.SkillPath + "/"
		if selected.Source.SkillPath == "." {
			prefix = "repo/"
		}
		entries = append(entries, zipEntry{prefix + path, files[path], 0755})
	}
	archive := zipBytes(t, entries...)
	urls := []string{}
	f.e.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		urls = append(urls, request.URL.String())
		if request.URL.String() == "https://api.github.com/repos/owner/repo/commits/"+selected.Source.Revision {
			return response(request, 200, []byte(`{"sha":"`+selected.Source.Revision+`"}`)), nil
		}
		if request.URL.String() != "https://codeload.github.com/owner/repo/zip/"+selected.Source.Revision {
			t.Fatalf("unexpected network request: %s", request.URL)
		}
		return response(request, 200, archive), nil
	})}
	return &urls
}

func TestPreparePreviewApplyAndRecovery(t *testing.T) {
	f := setup(t)
	marker := filepath.Join(f.root, "executed")
	files := map[string]string{"SKILL.md": strings.Replace(skillManifest("example"), "---\nVerify", "custom-field: preserve-me\n---\nVerify", 1), "scripts/run.sh": "#!/bin/sh\ntouch " + marker + "\n", "references/usage.md": "Review the report."}
	selected := adoptionResult(t, files)
	urls := mockPackage(t, f, selected, files)
	destination := filepath.Join(f.root, "installed", "example")
	state := filepath.Join(f.root, "state")
	plan, err := f.e.PrepareAdoption(resultJSON(t, selected), 1, destination, "codex", state, f.root, []string{filepath.Dir(destination)})
	must(t, err)
	if plan["status"] != "review_required" || plan["applied"] != false || plan["executed"] != false || exists(destination) || exists(marker) || len(*urls) != 2 {
		t.Fatal(plan, urls)
	}
	quarantine := str(plan["quarantine"])
	digest := str(plan["promotion"].(M)["tree_sha256"])
	reviewDigest := str(plan["promotion"].(M)["review_sha256"])
	_, err = f.e.Promote(quarantine, destination, "", false, true)
	reject(t, err)
	_, err = f.e.Promote(quarantine, destination, reviewDigest, false, false)
	must(t, err)
	if exists(destination) {
		t.Fatal("preview wrote installation")
	}
	applied, err := f.e.Promote(quarantine, destination, reviewDigest, false, true)
	must(t, err)
	if applied["applied"] != true || exists(marker) {
		t.Fatal(applied)
	}
	for path, expected := range files {
		actual, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(path)))
		must(t, err)
		if string(actual) != expected {
			t.Fatalf("installed bytes differ for %s", path)
		}
	}
	_, err = f.e.Promote(quarantine, destination, reviewDigest, false, true)
	reject(t, err)
	preview, err := f.e.Archive([]string{destination}, state, false, "")
	must(t, err)
	if preview["applied"] != false || !exists(destination) {
		t.Fatal(preview)
	}
	archived, err := f.e.Archive([]string{destination}, state, true, "")
	must(t, err)
	if exists(destination) {
		t.Fatal("archive did not remove installation from discovery")
	}
	id := str(archived["id"])
	_, err = f.e.Restore(id, state, false)
	must(t, err)
	if exists(destination) {
		t.Fatal("restore preview wrote installation")
	}
	_, err = f.e.Restore(id, state, true)
	must(t, err)
	actual, err := f.e.treeDigest(destination)
	must(t, err)
	if actual != digest || exists(marker) {
		t.Fatal("recovery changed reviewed contents or executed a script")
	}
	plan, err = f.e.PrepareAdoption(resultJSON(t, selected), 1, destination, "codex", state, f.root, []string{filepath.Dir(destination)})
	must(t, err)
	if plan["status"] != "destination_exists" || len(plan["installed"].([]M)) != 1 || plan["installed"].([]M)[0]["same_manifest"] != true || len(*urls) != 2 {
		t.Fatal("existing local installation was not detected without a fetch", plan)
	}
}

func TestPrepareRejectsUntrustedSelectionBeforeNetwork(t *testing.T) {
	mutations := map[string]func(*selectedSkill){
		"revision":   func(s *selectedSkill) { s.Source.Revision = "main" },
		"repository": func(s *selectedSkill) { s.Source.Repository = "../../outside" },
		"path":       func(s *selectedSkill) { s.Source.SkillPath = "skills/../outside" },
		"source-url": func(s *selectedSkill) { s.Source.URL = strings.Replace(s.Source.URL, "owner/repo", "other/repo", 1) },
		"url-revision": func(s *selectedSkill) {
			s.Source.URL = strings.Replace(s.Source.URL, s.Source.Revision, strings.Repeat("b", 40), 1)
		},
		"url-query":       func(s *selectedSkill) { s.Source.URL += "?private=true" },
		"metadata-only":   func(s *selectedSkill) { s.AdoptionStatus = "metadata_only" },
		"unavailable":     func(s *selectedSkill) { s.AdoptionStatus = "unavailable" },
		"identity":        func(s *selectedSkill) { s.SkillID = "other/repo/skills/example" },
		"file-path":       func(s *selectedSkill) { s.Source.Files[0].Path = "../outside" },
		"file-digest":     func(s *selectedSkill) { s.Source.Files[0].GitBlobSHA1 = "bad" },
		"duplicate-file":  func(s *selectedSkill) { s.Source.Files = append(s.Source.Files, s.Source.Files[0]) },
		"manifest-digest": func(s *selectedSkill) { s.Source.ManifestSHA256 = "bad" },
		"incompatible": func(s *selectedSkill) {
			s.HarnessSupport = []HarnessSupport{{Harness: "codex", Status: "incompatible"}}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			f := setup(t)
			selected := adoptionResult(t, map[string]string{"SKILL.md": skillManifest("example")})
			mutate(&selected)
			f.e.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("invalid result reached the network")
				return nil, nil
			})}
			_, err := f.e.PrepareAdoption(resultJSON(t, selected), 1, filepath.Join(f.root, "installed", "example"), "codex", filepath.Join(f.root, "state"), f.root, []string{})
			reject(t, err)
			if exists(filepath.Join(f.root, "state")) {
				t.Fatal("invalid selection mutated state")
			}
		})
	}
}

func TestGitBlobDigestIncludesHeader(t *testing.T) {
	if actual := gitBlobSHA1([]byte("hello\n")); actual != "ce013625030ba8dba906f756967f9e9ca394464a" {
		t.Fatal(actual)
	}
}

func TestPrepareRejectsPackageDrift(t *testing.T) {
	for _, variation := range []string{"manifest", "support", "extra", "missing", "ignored-extra"} {
		t.Run(variation, func(t *testing.T) {
			f := setup(t)
			files := map[string]string{"SKILL.md": skillManifest("example"), "references/usage.md": "Expected reference"}
			selected := adoptionResult(t, files)
			switch variation {
			case "manifest":
				files["SKILL.md"] += "Changed instructions"
			case "support":
				files["references/usage.md"] = "Changed reference"
			case "extra":
				files["scripts/unreviewed.sh"] = "exit 0"
			case "missing":
				delete(files, "references/usage.md")
			case "ignored-extra":
				files[".git/config"] = "extra"
			}
			mockPackage(t, f, selected, files)
			state := filepath.Join(f.root, "state")
			_, err := f.e.PrepareAdoption(resultJSON(t, selected), 1, filepath.Join(f.root, "installed", "example"), "codex", state, f.root, []string{})
			reject(t, err)
			entries, err := os.ReadDir(filepath.Join(state, "quarantine"))
			must(t, err)
			if len(entries) != 0 {
				t.Fatal("invalid package left a promotable quarantine")
			}
		})
	}
}

func TestPrepareRejectsMismatchedPinnedCommit(t *testing.T) {
	f := setup(t)
	selected := adoptionResult(t, map[string]string{"SKILL.md": skillManifest("example")})
	calls := 0
	f.e.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		return response(request, 200, []byte(`{"sha":"`+strings.Repeat("b", 40)+`"}`)), nil
	})}
	_, err := f.e.PrepareAdoption(resultJSON(t, selected), 1, filepath.Join(f.root, "installed", "example"), "codex", filepath.Join(f.root, "state"), f.root, []string{})
	reject(t, err)
	if calls != 1 {
		t.Fatal("downloaded a package after a revision mismatch")
	}
}

func TestPromotionRejectsPreparedMutation(t *testing.T) {
	for _, variation := range []string{"fingerprint", "package", "source-binding", "coherent-source-mutation", "different-destination"} {
		t.Run(variation, func(t *testing.T) {
			f := setup(t)
			files := map[string]string{"SKILL.md": skillManifest("example"), "references/usage.md": "Expected reference"}
			selected := adoptionResult(t, files)
			mockPackage(t, f, selected, files)
			destination := filepath.Join(f.root, "installed", "example")
			plan, err := f.e.PrepareAdoption(resultJSON(t, selected), 1, destination, "codex", filepath.Join(f.root, "state"), f.root, []string{})
			must(t, err)
			quarantine := str(plan["quarantine"])
			digest := str(plan["promotion"].(M)["review_sha256"])
			switch variation {
			case "fingerprint":
				digest = strings.Repeat("b", 64)
			case "package":
				writeFile(t, filepath.Join(quarantine, "skill", "references", "usage.md"), "Changed reference")
			case "source-binding":
				manifest := M{}
				must(t, readJSON(filepath.Join(quarantine, "manifest.json"), &manifest))
				manifest["commit"] = strings.Repeat("b", 40)
				must(t, atomicJSON(filepath.Join(quarantine, "manifest.json"), manifest))
			case "coherent-source-mutation":
				manifest := M{}
				must(t, readJSON(filepath.Join(quarantine, "manifest.json"), &manifest))
				selected.Source.Repository = "other/repo"
				selected.Source.URL = strings.Replace(selected.Source.URL, "owner/repo", "other/repo", 1)
				selected.SkillID = "other/repo/skills/example"
				manifest["catalog_result"], manifest["repository"] = selected, "other/repo"
				must(t, atomicJSON(filepath.Join(quarantine, "manifest.json"), manifest))
			case "different-destination":
				destination = filepath.Join(f.root, "different", "example")
			}
			_, err = f.e.Promote(quarantine, destination, digest, false, true)
			reject(t, err)
			if exists(destination) {
				t.Fatal("mutated review installed")
			}
		})
	}
}

func TestPrepareLegacyAndRepositoryRoot(t *testing.T) {
	for _, directory := range []string{"skills/example", "."} {
		f := setup(t)
		files := map[string]string{"SKILL.md": skillManifest("example"), "references/usage.md": "Reference"}
		selected := adoptionResult(t, files)
		selected.Source.Files = nil
		selected.Source.SkillPath = directory
		selected.SkillID = selected.Source.Repository + "/" + directory
		selected.Source.URL = "https://github.com/owner/repo/tree/" + selected.Source.Revision + "/" + directory
		mockPackage(t, f, selected, files)
		plan, err := f.e.PrepareAdoption(resultJSON(t, selected), 1, filepath.Join(f.root, "installed", "example"), "codex", filepath.Join(f.root, "state"), f.root, []string{})
		must(t, err)
		if strings.Contains(str(plan["package_integrity"]), "complete") {
			t.Fatal("legacy result claimed complete package integrity")
		}
	}
}

func TestPreparePreservesExistingDestination(t *testing.T) {
	f := setup(t)
	selected := adoptionResult(t, map[string]string{"SKILL.md": skillManifest("example")})
	destination := filepath.Join(f.root, "installed", "example")
	writeFile(t, filepath.Join(destination, "keep.txt"), "Preexisting user data")
	plan, err := f.e.PrepareAdoption(resultJSON(t, selected), 1, destination, "codex", filepath.Join(f.root, "state"), f.root, []string{})
	must(t, err)
	data, err := os.ReadFile(filepath.Join(destination, "keep.txt"))
	must(t, err)
	if plan["status"] != "destination_exists" || string(data) != "Preexisting user data" || exists(filepath.Join(f.root, "state")) {
		t.Fatal(plan)
	}
}

func TestPreparedPromotionPreservesRawManifest(t *testing.T) {
	for _, manifest := range []string{"\ufeff" + skillManifest("example"), strings.ReplaceAll(skillManifest("example"), "\n", "\r\n")} {
		f := setup(t)
		files := map[string]string{"SKILL.md": manifest}
		selected := adoptionResult(t, files)
		mockPackage(t, f, selected, files)
		destination := filepath.Join(f.root, "installed", "example")
		plan, err := f.e.PrepareAdoption(resultJSON(t, selected), 1, destination, "codex", filepath.Join(f.root, "state"), f.root, []string{})
		must(t, err)
		_, err = f.e.Promote(str(plan["quarantine"]), destination, str(plan["promotion"].(M)["review_sha256"]), false, true)
		must(t, err)
		actual, err := os.ReadFile(filepath.Join(destination, "SKILL.md"))
		must(t, err)
		if string(actual) != manifest {
			t.Fatal("promotion rewrote the reviewed manifest bytes")
		}
	}
}

func TestPrepareRejectsQuarantineInExplicitRoot(t *testing.T) {
	f := setup(t)
	files := map[string]string{"SKILL.md": skillManifest("example")}
	selected := adoptionResult(t, files)
	urls := mockPackage(t, f, selected, files)
	customRoot := filepath.Join(f.root, "custom-skills")
	_, err := f.e.PrepareAdoption(resultJSON(t, selected), 1,
		filepath.Join(customRoot, "example"), "codex", filepath.Join(customRoot, "pending"), f.root, []string{customRoot})
	reject(t, err)
	if len(*urls) != 0 || exists(customRoot) {
		t.Fatal("quarantine inside an explicit root reached network or staging")
	}
}
