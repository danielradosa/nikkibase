import { defineRailway, project, service } from "railway/iac";

export const partial = "nikkibase";

export default defineRailway(() => {
  const nikkibase = service("nikkibase", {
    build: { builder: "DOCKERFILE", dockerfilePath: "Dockerfile" },
    deploy: { restartPolicyType: "ON_FAILURE", restartPolicyMaxRetries: 3 },
  });
  return project("nikkibase", {
    resources: [nikkibase],
  });
});
