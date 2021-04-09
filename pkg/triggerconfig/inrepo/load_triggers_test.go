package inrepo

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/lighthouse-client/pkg/config/job"

	"github.com/h2non/gock"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse-client/pkg/config"
	"github.com/jenkins-x/lighthouse-client/pkg/filebrowser"
	fbfake "github.com/jenkins-x/lighthouse-client/pkg/filebrowser/fake"
	"github.com/jenkins-x/lighthouse-client/pkg/plugins"
	"github.com/jenkins-x/lighthouse-client/pkg/scmprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeConfig(t *testing.T) {
	owner := "myorg"
	repo := "loadtest"
	ref := "master"
	fullName := scm.Join(owner, repo)

	scmClient, fakeData := fake.NewDefault()
	scmProvider := scmprovider.ToClient(scmClient, "my-bot")

	fakeData.Repositories = []*scm.Repository{
		{
			Namespace: owner,
			Name:      repo,
			FullName:  fullName,
			Branch:    "master",
		},
	}

	fileBrowsers, err := filebrowser.NewFileBrowsers(filebrowser.GitHubURL, filebrowser.NewFileBrowserFromScmClient(scmProvider))
	require.NoError(t, err, "failed to create filebrowsers")

	cfg := &config.Config{}
	pluginCfg := &plugins.Configuration{}
	flag, err := MergeTriggers(cfg, pluginCfg, fileBrowsers, NewResolverCache(), owner, repo, ref)
	require.NoError(t, err, "failed to merge configs")
	assert.True(t, flag, "did not return merge flag")

	LogConfig(t, cfg)

	r := owner + "/" + repo
	assert.Len(t, cfg.Presubmits[r], 2, "presubmits for repo %s", r)
	assert.Len(t, cfg.Postsubmits[r], 1, "postsubmits for repo %s", r)
}

func TestInvalidConfigs(t *testing.T) {
	scmClient, _ := fake.NewDefault()
	scmProvider := scmprovider.ToClient(scmClient, "my-bot")

	fileBrowsers, err := filebrowser.NewFileBrowsers(filebrowser.GitHubURL, filebrowser.NewFileBrowserFromScmClient(scmProvider))
	require.NoError(t, err, "failed to create filebrowsers")

	invalidRepos := []string{"duplicate-presubmit", "duplicate-postsubmit"}
	for _, repo := range invalidRepos {
		owner := "myorg"
		ref := "master"
		_, err := LoadTriggerConfig(fileBrowsers, NewResolverCache(), owner, repo, ref)
		require.Errorf(t, err, "should have failed to load triggers from repo %s/%s with ref %s", owner, repo, ref)

		t.Logf("got expected error loading invalid configuration on repo %s of: %s", repo, err.Error())
	}
}

func TestEmptyDirectoryDoesNotFail(t *testing.T) {
	fileBrowsers, err := filebrowser.NewFileBrowsers(filebrowser.GitHubURL, fbfake.NewFakeFileBrowser(filepath.Join("test_data", "empty_dir")))
	require.NoError(t, err, "failed to create filebrowsers")

	owner := "myorg"
	repo := "myrepo"
	ref := "master"
	config, err := LoadTriggerConfig(fileBrowsers, NewResolverCache(), owner, repo, ref)
	require.NoErrorf(t, err, "should not fail to load triggers for repo %s/%s with ref %s", owner, repo, ref)
	require.NotNil(t, config, "no config for repo %s/%s with ref %s", owner, repo, ref)

	assert.Empty(t, config.Spec.Presubmits, "should have no presubmits")
	assert.Empty(t, config.Spec.Postsubmits, "should have no postsubmits")
}

func TestLoadJobFromURL(t *testing.T) {
	defer gock.Off()

	gock.New("https://raw.githubusercontent.com").
		Get("/rawlingsj/test/master/foo.yaml").
		Reply(200).
		Type("application/json").
		File("test_data/load_url/foo.yaml")

	j := &job.Base{}
	err := loadJobBaseFromSourcePath(nil, NewResolverCache(), j, "", "", "https://raw.githubusercontent.com/rawlingsj/test/master/foo.yaml", "")
	assert.NoError(t, err, "should not have an error returned")
	assert.Equal(t, "jenkinsxio/chuck:0.0.1", j.PipelineRunSpec.PipelineSpec.Tasks[0].TaskSpec.Steps[0].Image, "image name for task is not correct")
}
