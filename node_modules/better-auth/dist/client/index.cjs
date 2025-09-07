'use strict';

const vanilla = require('../shared/better-auth.QrcI0Z8z.cjs');
const query = require('../shared/better-auth.CYZmwTIS.cjs');
require('../shared/better-auth.fOOSrFgz.cjs');
require('@better-fetch/fetch');
require('../shared/better-auth.DRmln2Nr.cjs');
require('../shared/better-auth.B6fIklBU.cjs');
require('../shared/better-auth.ANpbi45u.cjs');
require('nanostores');
require('../shared/better-auth.DhsGZ30Q.cjs');
require('../shared/better-auth.BLDOwz3i.cjs');

const InferPlugin = () => {
  return {
    id: "infer-server-plugin",
    $InferServerPlugin: {}
  };
};
function InferAuth() {
  return {};
}

exports.createAuthClient = vanilla.createAuthClient;
exports.useAuthQuery = query.useAuthQuery;
exports.InferAuth = InferAuth;
exports.InferPlugin = InferPlugin;
