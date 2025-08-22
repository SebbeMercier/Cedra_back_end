// middleware/authMiddleware.js
import jwt from "jsonwebtoken";

export const authMiddleware = (req, res, next) => {
  const hdr = req.headers.authorization || "";
  if (!hdr.startsWith("Bearer ")) {
    return res.status(401).json({ message: "Non autorisé, token manquant" });
  }
  const token = hdr.slice(7);
  try {
    req.user = jwt.verify(token, process.env.JWT_SECRET); // { id, email, companyId, ... }
    next();
  } catch (e) {
    return res.status(401).json({ message: "Token invalide" });
  }
};
