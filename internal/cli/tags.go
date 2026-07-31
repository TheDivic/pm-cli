package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// tagCount is one tag and the number of tasks that use it.
type tagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func newTagsCmd(opts *GlobalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "tags",
		Short: "List all tags in use across projects, with usage counts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			refs, _, err := loadRefs(opts)
			if err != nil {
				return err
			}
			counts := map[string]int{}
			for _, r := range refs {
				for _, tag := range r.Task.Tags {
					counts[tag]++
				}
			}
			list := make([]tagCount, 0, len(counts))
			for tag, n := range counts {
				list = append(list, tagCount{Tag: tag, Count: n})
			}
			// Most-used first, then alphabetical for stable output.
			sort.Slice(list, func(i, j int) bool {
				if list[i].Count != list[j].Count {
					return list[i].Count > list[j].Count
				}
				return list[i].Tag < list[j].Tag
			})

			if opts.JSON {
				return writeTagsJSON(cmd.OutOrStdout(), list)
			}
			writeTagsText(cmd.OutOrStdout(), list)
			return nil
		},
	}
}

func writeTagsJSON(w io.Writer, list []tagCount) error {
	out := struct {
		Tags []tagCount `json:"tags"`
	}{Tags: list}
	if out.Tags == nil {
		out.Tags = []tagCount{}
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeTagsText(w io.Writer, list []tagCount) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "TAG\tCOUNT")
	for _, tc := range list {
		fmt.Fprintf(tw, "%s\t%d\n", tc.Tag, tc.Count)
	}
	_ = tw.Flush()
}
