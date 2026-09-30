import { defineRailway, preserve, project, service, volume } from "railway/iac";

export const partial = "nikkibase";

export default defineRailway(() => {
  const pushes = volume("nikkibase-push");
  const nikkibase = service("nikkibase", {
    build: { builder: "DOCKERFILE", dockerfilePath: "Dockerfile" },
    deploy: { restartPolicyType: "ON_FAILURE", restartPolicyMaxRetries: 3 },
    volumeMounts: { "/data": pushes },
    variables: {
      PUSH_DIR: "/data/push",
      RAILWAY_RUN_UID: "0",
      VAPID_PRIVATE_KEY: preserve(),
    },
  });
  return project("nikkibase", {
    resources: [nikkibase, pushes],
  });
});
