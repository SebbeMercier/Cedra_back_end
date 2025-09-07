import express from "express";
import Stripe from "stripe";
import dotenv from "dotenv";

dotenv.config();

const router = express.Router();

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY, {
  apiVersion: "2024-04-10",
});

router.post("/create-payment-intent", async (req, res) => {
  try {
    const { amount } = req.body;

    // ✅ Validation du montant (en centimes)
    if (!amount || typeof amount !== "number" || amount < 50) {
      return res.status(400).json({ error: "Montant invalide ou trop bas (min. 50 centimes)" });
    }

    // ✅ Création du PaymentIntent pour carte uniquement
    const paymentIntent = await stripe.paymentIntents.create({
  amount: Math.round(amount), // ex: 1099
  currency: "eur",
  payment_method_types: ['card']
});


    // ✅ Envoi du client secret
    res.status(200).json({ clientSecret: paymentIntent.client_secret });
  } catch (error) {
    console.error("❌ Erreur Stripe :", error);
    res.status(500).json({ error: error.message || "Erreur interne Stripe" });
  }
});

export default router;
