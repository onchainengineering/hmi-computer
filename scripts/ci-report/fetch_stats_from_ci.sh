#!/usr/bin/env bash
set -euo pipefail

# Usage: ./fetch_stats_from_ci.sh
#
# This script is for fetching historic test stats from GitHub Actions CI.
#
# Requires gh with credentials.
#
# https://github.com/cli/cli/blob/trunk/pkg/cmd/run/view/view.go#L434

dir="$(dirname "$0")"/ci-stats
mkdir -p "${dir}"

pushd "${dir}" >/dev/null

# Stats step name, used for filtering log.
job_step_name="Print test stats"

if [[ ! -f list-ci.yaml.json ]]; then
	gh run list -w ci.yaml -L 1000 --json conclusion,createdAt,databaseId,displayTitle,event,headBranch,headSha,name,number,startedAt,status,updatedAt,url,workflowDatabaseId,workflowName \
		>list-ci.yaml.json || {
		rm -f list-ci.yaml.json
		exit 1
	}
fi

runs="$(
	jq -r '.[] | select(.status == "completed") | select(.conclusion == "success" or .conclusion == "failure") | [.databaseId, .event, .displayTitle, .headBranch, .headSha, .url] | @tsv' \
		<list-ci.yaml.json
)"

while read -r run; do
	mapfile -d $'\t' -t parts <<<"${run}"
	parts[-1]="${parts[-1]%$'\n'}"

	database_id="${parts[0]}"
	event="${parts[1]}"
	display_title="${parts[2]}"
	head_branch="${parts[3]}"
	head_sha="${parts[4]}"
	run_url="${parts[5]}"

	# Check if this run predates the stats PR, if yes, skip it:
	# https://github.com/coder/coder/issues/6676
	if [[ ${database_id} -le 4595490577 ]]; then
		echo "Skipping ${database_id} (${display_title}), too old..."
		continue
	fi

	run_jobs_file=run-"${database_id}"-"${event}"-jobs.json
	if [[ ! -f "${run_jobs_file}" ]]; then
		echo "Fetching jobs for run: ${display_title} (${database_id}, ${event}, ${head_branch})"
		gh run view "${database_id}" --json jobs >"${run_jobs_file}" || {
			rm -f "${run_jobs_file}"
			exit 1
		}
	fi

	jobs="$(
		jq -r '.jobs[] | select(.name | startswith("test-go")) | select(.status == "completed") | select(.conclusion == "success" or .conclusion == "failure") | [.databaseId, .startedAt, .completedAt, .name, .url] | @tsv' \
			<"${run_jobs_file}"
	)"

	while read -r job; do
		mapfile -d $'\t' -t parts <<<"${job}"
		parts[-1]="${parts[-1]%$'\n'}"

		job_database_id="${parts[0]}"
		job_started_at="${parts[1]}"
		job_completed_at="${parts[2]}"
		job_name="${parts[3]}"
		job_url="${parts[4]}"

		job_log=run-"${database_id}"-job-"${job_database_id}"-"${job_name}".log
		if [[ ! -f "${job_log}" ]]; then
			echo "Fetching log for: ${job_name} (${job_database_id}, ${job_url})"
			# Example log (partial).
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:18.4063489Z ##[group]Run # Artifacts are not available after rerunning a job,
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:18.4063872Z # Artifacts are not available after rerunning a job,
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:18.4064188Z # so we need to print the test stats to the log.
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:18.4064642Z go run ./scripts/ci-report/main.go gotests.json | tee gotests_stats.json
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:18.4110112Z shell: /usr/bin/bash -e {0}
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:18.4110364Z ##[endgroup]
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:19.3440469Z {
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:19.3441078Z   "packages": [
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:19.3441448Z     {
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:19.3442927Z       "name": "agent",
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:19.3443311Z       "time": 17.538
			#   test-go (ubuntu-latest)	Print test stats	2023-04-11T03:02:19.3444048Z     },
			#   ...
			gh run view --job "${job_database_id}" --log >"${job_log}" || {
				# Sometimes gh fails to extract ZIP, etc. :'(
				rm -f "${job_log}"
				echo "Failed to fetch log for: ${job_name} (${job_database_id}, ${job_url}), skipping..."
				continue
			}
			log_lines="$(wc -l "${job_log}" | awk '{print $1}')"
			if [[ ${log_lines} -lt 2 ]]; then
				# Sometimes gh returns nothing and gives no error :'(
				rm -f "${job_log}"
				echo "Log is empty for: ${job_name} (${job_database_id}, ${job_url}), skipping..."
				continue
			fi
		fi

		if ! job_stats="$(
			# Extract the stats job output (JSON) from the job log,
			# discarding the timestamp and non-JSON header.
			#
			# Example variable values:
			#   job_name="test-go (ubuntu-latest)"
			#   job_step_name="Print test stats"
			grep "${job_name}.*${job_step_name}" "${job_log}" |
				sed -E 's/.*[0-9-]{10}T[0-9:]{8}\.[0-9]*Z //' |
				grep -E "^[{}\ ].*"
		)"; then
			echo "Failed to find stats in job log: ${job_name} (${job_database_id}, ${job_url}), skipping..."
			continue
		fi

		if ! jq -e . >/dev/null 2>&1 <<<"${job_stats}"; then
			# Sometimes actions logs are partial when fetched via CLI :'(
			echo "Failed to parse stats for: ${job_name} (${job_database_id}, ${job_url}), skipping..."
			continue
		fi

		job_stats_file=run-"${database_id}"-job-"${job_database_id}"-"${job_name}"-stats.json
		if [[ -f "${job_stats_file}" ]]; then
			continue
		fi
		jq \
			--argjson run_id "${database_id}" \
			--arg run_url "${run_url}" \
			--arg event "${event}" \
			--arg branch "${head_branch}" \
			--arg sha "${head_sha}" \
			--arg started_at "${job_started_at}" \
			--arg completed_at "${job_completed_at}" \
			--arg display_title "${display_title}" \
			--argjson job_id "${job_database_id}" \
			--arg job "${job_name}" \
			--arg job_url "${job_url}" \
			'{run_id: $run_id, run_url: $run_url, event: $event, branch: $branch, sha: $sha, started_at: $started_at, completed_at: $completed_at, display_title: $display_title, job_id: $job_id, job: $job, job_url: $job_url, stats: .}' \
			<<<"${job_stats}" \
			>"${job_stats_file}" || {
			echo "Failed to write stats for: ${job_name} (${job_database_id}, ${job_url}), skipping..."
			rm -f "${job_stats_file}"
			exit 1
		}
	done <<<"${jobs}"
done <<<"${runs}"
