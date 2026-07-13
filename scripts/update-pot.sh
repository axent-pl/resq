#!/usr/bin/env bash

set -euo pipefail

readonly script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly project_root="$(cd "${script_dir}/.." && pwd)"
readonly output="${project_root}/cmd/resq/i18n/locales/resq.pot"

if ! command -v xgettext >/dev/null 2>&1; then
	echo "error: xgettext is required (install GNU gettext)" >&2
	exit 1
fi

readonly work_dir="$(mktemp -d "${TMPDIR:-/tmp}/resq-xgettext.XXXXXX")"
readonly scan_root="${work_dir}/source"
readonly generated_pot="${work_dir}/resq.pot"
trap 'rm -rf "${work_dir}"' EXIT

mkdir -p "${scan_root}"
sources=()

# Copy Go sources into the temporary tree so POT references stay relative to
# the repository root rather than containing absolute machine-specific paths.
while IFS= read -r -d '' source; do
	relative="${source#${project_root}/}"
	mkdir -p "${scan_root}/$(dirname "${relative}")"
	cp "${source}" "${scan_root}/${relative}"
	sources+=("${relative}")
done < <(
	find "${project_root}" \
		-type d \( -name .git -o -name vendor \) -prune -o \
		-type f -name '*.go' -print0
)

# Go templates are not a native xgettext input format. Convert literal calls
# such as {{ _ .Lang "Events" }} into one Mark("Events") call on the same
# line. Dynamic values must be wrapped with i18n.Mark at their Go definition.
while IFS= read -r -d '' source; do
	relative="${source#${project_root}/}"
	generated="${relative}.go"
	mkdir -p "${scan_root}/$(dirname "${generated}")"

	awk '
		{
			pattern = "\\{\\{[[:space:]]*_[[:space:]]+[^[:space:]]+[[:space:]]+\\\""
			line = $0
			calls = ""
			while (match(line, pattern)) {
				message = substr(line, RSTART + RLENGTH)
				closing_quote = 0
				escaped = 0
				for (i = 1; i <= length(message); i++) {
					character = substr(message, i, 1)
					if (!escaped && character == "\"") {
						closing_quote = i
						break
					}
					if (!escaped && character == "\\") {
						escaped = 1
					} else {
						escaped = 0
					}
				}
				if (closing_quote == 0) {
					break
				}
				calls = calls "Mark(\"" substr(message, 1, closing_quote - 1) "\"); "
				line = substr(message, closing_quote + 1)
			}
			print calls
		}
	' "${source}" >"${scan_root}/${generated}"

	sources+=("${generated}")
done < <(
	find "${project_root}" \
		-type d \( -name .git -o -name vendor \) -prune -o \
		-type f \( -name '*.html' -o -name '*.tmpl' -o -name '*.gotmpl' -o -name '*.gohtml' \) -print0
)

if (( ${#sources[@]} == 0 )); then
	echo "error: no Go or Go template sources found" >&2
	exit 1
fi

(
	cd "${scan_root}"
	xgettext \
		--language=Go \
		--from-code=UTF-8 \
		--keyword \
		--keyword=Mark:1 \
		--add-comments=TRANSLATORS: \
		--sort-by-file \
		--package-name=ResQ \
		--output="${generated_pot}" \
		"${sources[@]}"
)

mkdir -p "$(dirname "${output}")"

# Restore the original template filenames in source references and update the
# catalog atomically only after extraction succeeds.
sed 's/\.html\.go:/.html:/g; s/\.tmpl\.go:/.tmpl:/g; s/\.gotmpl\.go:/.gotmpl:/g; s/\.gohtml\.go:/.gohtml:/g' \
	"${generated_pot}" >"${output}.tmp"
mv "${output}.tmp" "${output}"

echo "updated ${output#${project_root}/}"
