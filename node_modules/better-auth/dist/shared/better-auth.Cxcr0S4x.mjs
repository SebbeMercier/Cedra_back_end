import { defaultRoles } from '../plugins/organization/access/index.mjs';
import * as z from 'zod/v4';
import { APIError } from 'better-call';
import './better-auth.CewjboYP.mjs';
import './better-auth.DV5EHeYG.mjs';
import './better-auth.CMQ3rA-I.mjs';
import '@better-auth/utils/base64';
import '@better-auth/utils/hmac';
import './better-auth.BjBlybv-.mjs';
import '@better-auth/utils/binary';
import './better-auth.Dcv8PS7T.mjs';

let cacheAllRoles = /* @__PURE__ */ new Map();
const hasPermission = async (input, ctx) => {
  let acRoles = { ...input.options.roles || defaultRoles };
  if (ctx && input.organizationId && input.options.dynamicAccessControl?.enabled && input.options.ac && !input.useMemoryCache) {
    const roles = await ctx.context.adapter.findMany({
      model: "organizationRole",
      where: [
        {
          field: "organizationId",
          value: input.organizationId
        }
      ]
    });
    for (const { role, permission: permissionsString } of roles) {
      if (role in acRoles) continue;
      const result = z.record(z.string(), z.array(z.string())).safeParse(JSON.parse(permissionsString));
      if (!result.success) {
        ctx.context.logger.error(
          "[hasPermission] Invalid permissions for role " + role,
          {
            permissions: JSON.parse(permissionsString)
          }
        );
        throw new APIError("INTERNAL_SERVER_ERROR", {
          message: "Invalid permissions for role " + role
        });
      }
      acRoles[role] = input.options.ac.newRole(result.data);
    }
  }
  if (input.useMemoryCache) {
    acRoles = cacheAllRoles.get(input.organizationId) || acRoles;
  }
  cacheAllRoles.set(input.organizationId, acRoles);
  return hasPermissionFn(input, acRoles);
};
const clientSideHasPermission = async (input) => {
  const acRoles = input.options.roles || defaultRoles;
  return hasPermissionFn(input, acRoles);
};
const hasPermissionFn = (input, acRoles) => {
  if (!input.permissions && !input.permission) return false;
  const roles = input.role.split(",");
  const creatorRole = input.options.creatorRole || "owner";
  const isCreator = roles.includes(creatorRole);
  const allowCreatorsAllPermissions = input.allowCreatorAllPermissions || false;
  if (isCreator && allowCreatorsAllPermissions) return true;
  for (const role of roles) {
    const _role = acRoles[role];
    const result = _role?.authorize(input.permissions ?? input.permission);
    if (result?.success) {
      return true;
    }
  }
  return false;
};

export { clientSideHasPermission as c, hasPermission as h };
