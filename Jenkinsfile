def getAgentLabel() {
    def rawNode = params.DEPLOY_NODE ?: 'test'
    def nodeMap = [
        // 在 Jenkins 中为实际节点配置这些逻辑标签，避免在仓库暴露内网拓扑。
        'prod'  : 'ops-hub-prod',
        'test'  : 'ops-hub-test',
        'dev'   : 'ops-hub-dev'
    ]
    return nodeMap[rawNode] ?: rawNode
}

pipeline {
    agent {
        label getAgentLabel()
    }

    parameters {
        choice(name: 'DEPLOY_NODE', choices: ['test', 'dev', 'prod'], description: '部署目标节点')
        booleanParam(name: 'BUILD_SERVER', defaultValue: true, description: '是否重新构建后端并更新 Docker 镜像')
        booleanParam(name: 'BUILD_FRONTEND', defaultValue: true, description: '是否编译前端静态资源')
    }

    environment {
        TZ = 'Asia/Shanghai'
    }

    options {
        timeout(time: 30, unit: 'MINUTES')
        buildDiscarder(logRotator(numToKeepStr: '10'))
        timestamps()
    }

    stages {
        stage('Initialize') {
            steps {
                echo '========== 初始化构建环境 =========='
                script {
                    if (params.DEPLOY_NODE == 'prod') {
                        def jobName = env.JOB_NAME ?: ''
                        if (!jobName.contains('Prod')) {
                            error "部署失败: 只有任务名称中包含 'Prod' 的项目才允许选择 'prod' 节点进行部署！当前任务名称为: ${jobName}"
                        }
                    }
                }
                echo "构建参数: DEPLOY_NODE=${params.DEPLOY_NODE}, BUILD_SERVER=${params.BUILD_SERVER}, BUILD_FRONTEND=${params.BUILD_FRONTEND}"
                sh 'chmod +x builder/build.sh'
            }
        }

        stage('Build & Deploy') {
            steps {
                echo '========== 启动容器构建与部署 =========='
                script {
                    def serverParam = params.BUILD_SERVER ? '1' : '0'
                    def frontendParam = params.BUILD_FRONTEND ? '1' : '0'
                    sh "server=${serverParam} frontend=${frontendParam} ./builder/build.sh"
                }
            }
        }
    }

    post {
        success {
            echo '========== 部署成功 =========='
        }
        failure {
            echo '========== 部署失败，请检查构建日志 =========='
        }
    }
}
