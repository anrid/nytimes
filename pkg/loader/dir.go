package loader

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anrid/nytimes/pkg/domain"
	"github.com/pkg/errors"
)

type ReadDirWithArticlesParams struct {
	Path        string
	Suffix      string
	Verbose     bool
	StartFrom   string
	Max         int // If max > 0: Read max this many articles.
	EachArticle func(articlesTotal int, isLast bool, a *domain.NYTimesArticle) error
}

func ReadDirWithArticles(p ReadDirWithArticlesParams) error {
	var filesTotal int
	var articlesTotal int
	var startFromFileFound bool
	timer := time.Now()

	if p.Verbose {
		fmt.Printf("Reading dir: %s\n", p.Path)
	}

	des, err := os.ReadDir(p.Path)
	if err != nil {
		log.Fatal(err)
	}

	var files []string

	for _, de := range des {
		if !strings.HasSuffix(de.Name(), p.Suffix) {
			if p.Verbose {
				fmt.Printf("Skipping file: %s\n", de.Name())
			}
			continue
		} else if de.IsDir() {
			if p.Verbose {
				fmt.Printf("Skipping dir: %s\n", de.Name())
			}
			continue
		}

		files = append(files, de.Name())
	}

	sort.Slice(files, func(i, j int) bool {
		if len(files[i]) > 22 && len(files[j]) > 22 {
			// Special case for files named `articles-YYYY-M.json.gz`
			// where YYYY is a year and M is a 1 or 2 digit month.
			iYear := files[i][9:13]
			jYear := files[j][9:13]
			if iYear == jYear {
				if len(files[i]) == len(files[j]) {
					return files[i] < files[j]
				}
				return len(files[i]) < len(files[j])
			}
		}
		return files[i] < files[j]
	})

	for _, f := range files {
		if p.StartFrom != "" && !startFromFileFound {
			if !strings.Contains(f, p.StartFrom) {
				if p.Verbose {
					fmt.Printf("Skipping file: %s (starting from `%s`)\n", f, p.StartFrom)
				}
				continue
			}
			startFromFileFound = true
		}

		if p.Verbose {
			fmt.Printf("Reading file: %s\n", f)
		}

		f, err := os.Open(filepath.Join(p.Path, f))
		if err != nil {
			log.Fatal(err)
		}

		r, err := gzip.NewReader(f)
		if err != nil {
			log.Fatal(err)
		}

		data, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		nyt := new(domain.NYTimesMonthlyArticles)

		err = json.Unmarshal(data, nyt)
		if err != nil {
			log.Fatal(err)
		}

		filesTotal++

		for _, a := range nyt.Response.Docs {
			articlesTotal++

			// if p.Verbose && i == 0 {
			// 	util.Dump(a)
			// }

			if !strings.HasSuffix(a.PubDate, "+0000") {
				log.Fatalf("found unexpected data format `%s`", a.PubDate)
			}

			err = p.EachArticle(articlesTotal, false, a)
			if err != nil {
				return errors.Wrap(err, "got error when calling EachArticle function")
			}
		}

		r.Close()
		f.Close()

		if p.Max > 0 && articlesTotal >= p.Max {
			fmt.Printf("Read more than max articles (max: %d), exiting early!\n", p.Max)
			break
		}
	}

	// Make one final call passing `isLast: true` to allow indexers to
	// flush their buffers (if they use them).
	p.EachArticle(articlesTotal, true, nil)

	fmt.Printf("Done. Read %d files in %s\n", filesTotal, time.Since(timer))
	return nil
}
