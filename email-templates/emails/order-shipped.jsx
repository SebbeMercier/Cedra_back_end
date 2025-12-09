import React from 'react';
import {
  Html,
  Head,
  Preview,
  Body,
  Container,
  Section,
  Row,
  Column,
  Heading,
  Text,
  Button,
  Link,
  Hr,
  Tailwind,
} from '@react-email/components';

export default function OrderShippedEmail({
  orderID = '12345678',
  trackingNumber = 'FR1234567890',
  estimatedDelivery = '15 Décembre 2024',
  items = [
    { name: 'MacBook Pro M3', quantity: 1, price: 2499.99 },
    { name: 'Magic Mouse', quantity: 1, price: 99.99 },
  ],
}) {
  const total = items.reduce((sum, item) => sum + item.price * item.quantity, 0);

  return (
    <Tailwind>
      <Html lang="fr">
        <Head />
        <Preview>📦 Votre commande #{orderID} a été expédiée !</Preview>
        <Body className="bg-gray-50 font-sans">
          <Container className="max-w-2xl mx-auto my-10 bg-white rounded-2xl shadow-2xl overflow-hidden">
            
            {/* Header */}
            <Section className="bg-gradient-to-r from-blue-600 to-cyan-600 p-10 text-center">
              <Text className="text-6xl mb-4">📦</Text>
              <Heading className="text-white text-3xl font-bold m-0 mb-2">
                Votre commande est en route !
              </Heading>
              <Text className="text-white text-lg m-0 opacity-90">
                Suivez votre colis en temps réel
              </Text>
            </Section>

            {/* Content */}
            <Section className="px-8 py-10">
              <Text className="text-gray-800 text-lg leading-relaxed mb-6">
                Bonne nouvelle ! Votre commande a été expédiée et arrive bientôt chez vous. 🎉
              </Text>

              {/* Tracking Info */}
              <Section className="bg-gradient-to-br from-blue-50 to-cyan-50 p-6 rounded-xl border-2 border-blue-200 mb-8">
                <table width="100%" cellPadding="0" cellSpacing="0">
                  <tr>
                    <td width="50%" style={{ verticalAlign: 'top' }}>
                      <Text className="text-gray-600 text-sm font-semibold uppercase m-0 mb-1">
                        Numéro de suivi
                      </Text>
                      <Text className="bg-white px-3 py-2 rounded text-blue-600 text-base font-bold m-0" style={{ fontFamily: 'monospace' }}>
                        {trackingNumber}
                      </Text>
                    </td>
                    <td width="50%" style={{ textAlign: 'right', verticalAlign: 'top' }}>
                      <Text className="text-gray-600 text-sm font-semibold uppercase m-0 mb-1">
                        Livraison estimée
                      </Text>
                      <Text className="text-gray-900 text-lg font-bold m-0">
                        {estimatedDelivery}
                      </Text>
                    </td>
                  </tr>
                </table>
              </Section>

              {/* CTA Tracking */}
              <Section className="text-center mb-10">
                <Button
                  href={`http://cedra.eldocam.com:5173/tracking/${trackingNumber}`}
                  className="bg-gradient-to-r from-blue-600 to-cyan-600 text-white px-10 py-4 rounded-xl font-bold text-lg no-underline inline-block shadow-lg"
                >
                  🔍 Suivre mon colis
                </Button>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Order Details */}
              <Section className="mb-8">
                <Heading className="text-gray-900 text-xl font-bold mb-4">
                  📋 Détails de la commande
                </Heading>
                
                <Section className="bg-gray-50 p-6 rounded-xl">
                  <Text className="text-gray-600 text-sm m-0 mb-4">
                    Commande <strong className="text-gray-900">#{orderID}</strong>
                  </Text>

                  {/* Items List */}
                  <table width="100%" cellPadding="0" cellSpacing="0">
                    {items.map((item, index) => (
                      <tr key={index} style={{ borderBottom: index < items.length - 1 ? '1px solid #e5e7eb' : 'none' }}>
                        <td width="60%" style={{ padding: '12px 0' }}>
                          <Text className="text-gray-900 font-medium text-base m-0">
                            {item.name}
                          </Text>
                          <Text className="text-gray-500 text-sm m-0">
                            Quantité: {item.quantity}
                          </Text>
                        </td>
                        <td width="40%" style={{ textAlign: 'right', padding: '12px 0' }}>
                          <Text className="text-gray-900 font-semibold text-base m-0">
                            {(item.price * item.quantity).toFixed(2)}€
                          </Text>
                        </td>
                      </tr>
                    ))}
                  </table>

                  <Hr className="border-gray-300 my-4" />

                  {/* Total */}
                  <table width="100%" cellPadding="0" cellSpacing="0">
                    <tr>
                      <td width="60%">
                        <Text className="text-gray-900 font-bold text-lg m-0">
                          Total
                        </Text>
                      </td>
                      <td width="40%" style={{ textAlign: 'right' }}>
                        <Text className="text-blue-600 font-bold text-xl m-0">
                          {total.toFixed(2)}€
                        </Text>
                      </td>
                    </tr>
                  </table>
                </Section>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Delivery Timeline */}
              <Section className="mb-8">
                <Heading className="text-gray-900 text-xl font-bold mb-6">
                  🚚 Suivi de livraison
                </Heading>

                <Section className="mb-6">
                  <Text className="text-gray-900 font-semibold text-base m-0 mb-1">
                    ✅ Commande confirmée
                  </Text>
                  <Text className="text-gray-500 text-sm m-0 mb-4">
                    Votre paiement a été validé
                  </Text>
                </Section>

                <Section className="mb-6">
                  <Text className="text-gray-900 font-semibold text-base m-0 mb-1">
                    ✅ Colis préparé
                  </Text>
                  <Text className="text-gray-500 text-sm m-0 mb-4">
                    Votre commande a été emballée
                  </Text>
                </Section>

                <Section className="mb-6">
                  <Text className="text-blue-600 font-semibold text-base m-0 mb-1">
                    🚚 En cours de livraison
                  </Text>
                  <Text className="text-gray-500 text-sm m-0 mb-4">
                    Votre colis est en route
                  </Text>
                </Section>

                <Section>
                  <Text className="text-gray-400 font-semibold text-base m-0 mb-1">
                    📬 Livraison prévue
                  </Text>
                  <Text className="text-gray-400 text-sm m-0">
                    {estimatedDelivery}
                  </Text>
                </Section>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Help Section */}
              <Section className="bg-purple-50 p-6 rounded-xl border border-purple-200">
                <Text className="text-purple-900 font-semibold text-base m-0 mb-2">
                  💡 Besoin d'aide ?
                </Text>
                <Text className="text-purple-800 text-sm m-0 mb-4">
                  Notre équipe support est disponible 7j/7 pour répondre à vos questions.
                </Text>
                <Link
                  href="http://cedra.eldocam.com:5173/support"
                  className="text-purple-600 font-semibold text-sm no-underline"
                >
                  Contacter le support →
                </Link>
              </Section>
            </Section>

            {/* Footer */}
            <Section className="bg-gray-900 px-8 py-8 text-center">
              <Text className="text-gray-400 text-xs m-0 mb-2">
                © 2024 Cedra - Tous droits réservés
              </Text>
              <Text className="text-gray-500 text-xs m-0">
                <Link href="http://cedra.eldocam.com:5173" className="text-purple-400 no-underline">
                  Visiter notre site
                </Link>
                {' • '}
                <Link href="http://cedra.eldocam.com:5173/orders" className="text-purple-400 no-underline">
                  Mes commandes
                </Link>
              </Text>
            </Section>
          </Container>
        </Body>
      </Html>
    </Tailwind>
  );
}
