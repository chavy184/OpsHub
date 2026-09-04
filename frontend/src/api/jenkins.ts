import { http } from '@/lib/http'
import type { JenkinsJobInfo, JenkinsBuildInfo } from '@/types/api'

const PREFIX = '/api/v1/jenkins'

/** 获取 Jenkins Job 信息 */
export async function getJobInfo(job: string) {
  const res = await http.get<JenkinsJobInfo>(`${PREFIX}/job-info`, { job })
  return res.data
}

/** 获取最近构建历史 */
export async function getRecentBuilds(job: string, count = 10) {
  const res = await http.get<JenkinsBuildInfo[]>(`${PREFIX}/builds`, { job, count })
  return res.data
}

/** 获取单个构建详情 */
export async function getBuildInfo(job: string, number: number) {
  const res = await http.get<JenkinsBuildInfo>(`${PREFIX}/build-info`, { job, number })
  return res.data
}

/** 获取构建日志 */
export async function getConsoleOutput(job: string, number: number) {
  const res = await http.get<{ output: string }>(`${PREFIX}/console`, { job, number })
  return res.data.output
}
