<!-- This file contains helper functions useful across several classes/components -->

<script lang="ts">
export default {}

// GetNumberGroup tests that the given string has exactly one grouping of digits and returns that grouping.
// For instance, "123", "abc123", and "abc1xyz" are all acceptable.
//
// If more than 1 number group is found or none are found, the empty string is returned.
export function GetNumberGroup(s: string): string {
  let matches = s.match(/\d+/g)
  // juggling-check. JS  is insane: https://stackoverflow.com/a/28984306
  if (matches == null) return ''
  else if (matches.length != 1) return ''

  return matches[0]
}
</script>
