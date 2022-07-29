import { setOutput, setFailed } from '@actions/core';
import { getOctokit, context } from '@actions/github';
import { env } from 'process';
import { inc, clean, valid } from 'semver'
import { DateTime } from 'luxon';

const owner = 'CodeIntelligenceTesting'
const repo = 'cifuzz'
const octokit = getOctokit(env.INPUT_TOKEN || env.GITHUB_TOKEN)

const getLastVersion = async () => {
  try {
    // From the github api docs: 
    // "The latest release is the most recent non-prerelease, 
    // non-draft release, sorted by the created_at attribute"
    const { data } = await octokit.rest.repos.getLatestRelease({
      owner,
      repo
    });

    const lastVersion = valid(clean(data.tag_name))
    if (lastVersion == null) {
      setFailed(`invalid last version: ${data.tag_name}`)
    }

    console.log(`last version: ${lastVersion}`)
    return lastVersion

  } catch (error) {
    // no release by now
    if (error.status == 404) {
      return '0.0.0'
    }
    setFailed(`unexpected error: ${error.message}`)
  }
}

const run = async () => {
  try {
    const lastVersion = await getLastVersion()

    // default values for local execution
    let date = DateTime.now()
    let commit = "local"

    // context.payload is available when running in the github CI
    if (context.payload.head_commit) {
      date = DateTime.fromISO(context.payload.head_commit.timestamp)
      commit = context.payload.head_commit.id.substring(0, 12)
    }

    // following the go pseudo versions format 
    // https://go.dev/ref/mod#pseudo-versions
    const nextPatchVersion = inc(lastVersion, 'patch')
    const thisVersion = `v${nextPatchVersion}-0.${date.toFormat('yyyyLLddhhmmss')}-${commit}`
    if (valid(thisVersion) == null) {
      setFailed(`generated invalid version ${thisVersion}`)
      return

    }

    console.log(`generated version: ${thisVersion}`)
    setOutput('version', thisVersion);

  } catch (error) {
    setFailed(error.message);
  }
}

run()
