pipeline {
checkout([$class: 'GitSCM',
            branches: [[name: "master"]], 
            userRemoteConfigs: [[url: "${git@github.com:torosent/captureorderack.git}", credentialsId: '3f3274fa-9202-4f37-914f-91e9ae1bee06' ]]])
}
