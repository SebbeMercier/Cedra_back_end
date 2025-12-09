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
  Img,
  Tailwind,
  Font,
} from '@react-email/components';

export default function WelcomePremiumEmail({ userName = 'Utilisateur' }) {
  return (
    <Tailwind>
      <Html lang="fr">
        <Head>
          <Font
            fontFamily="Inter"
            fallbackFontFamily="Arial"
            webFont={{
              url: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap',
              format: 'woff2',
            }}
            fontWeight={400}
            fontStyle="normal"
          />
        </Head>
        <Preview>Bienvenue sur Cedra ! Profitez de 10% de réduction 🎁</Preview>
        <Body className="bg-gray-50 font-sans">
          <Container className="max-w-2xl mx-auto my-10 bg-white rounded-2xl shadow-2xl overflow-hidden">
            
            {/* Hero Header avec Image */}
            <Section className="relative bg-gradient-to-br from-purple-600 via-indigo-600 to-blue-600 p-0">
              <div className="absolute inset-0 bg-black opacity-10"></div>
              <Section className="relative z-10 text-center py-12 px-6">
                <Heading className="text-white text-4xl font-bold m-0 mb-3">
                  🎉 Bienvenue chez Cedra !
                </Heading>
                <Text className="text-white text-xl m-0 opacity-95">
                  Bonjour {userName}, prêt à découvrir nos produits ?
                </Text>
              </Section>
            </Section>

            {/* Main Content */}
            <Section className="px-8 py-10">
              <Text className="text-gray-800 text-lg leading-relaxed mb-6">
                Nous sommes ravis de vous accueillir dans la communauté <strong className="text-purple-600">Cedra</strong> ! 🛍️
              </Text>

              <Text className="text-gray-700 text-base leading-relaxed mb-8">
                Découvrez notre sélection exclusive de produits, profitez de nos offres spéciales et bénéficiez d'une expérience shopping unique.
              </Text>

              {/* CTA Principal */}
              <Section className="text-center my-10">
                <Button
                  href="http://cedra.eldocam.com:5173/products"
                  className="bg-gradient-to-r from-purple-600 to-indigo-600 text-white px-10 py-4 rounded-xl font-bold text-lg no-underline inline-block shadow-lg"
                >
                  🛍️ Découvrir nos produits
                </Button>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Benefits Grid */}
              <Section className="mb-8">
                <Heading className="text-gray-900 text-2xl font-bold mb-6 text-center">
                  🎁 Vos avantages exclusifs
                </Heading>

                <table width="100%" cellPadding="0" cellSpacing="0">
                  <tr>
                    <td width="50%" style={{ paddingRight: '8px', paddingBottom: '16px' }}>
                      <Section className="bg-gradient-to-br from-green-50 to-emerald-50 p-5 rounded-xl border border-green-200">
                        <Text className="text-4xl mb-2">🚚</Text>
                        <Text className="text-gray-900 font-bold text-base m-0 mb-1">
                          Livraison gratuite
                        </Text>
                        <Text className="text-gray-600 text-sm m-0">
                          Dès 50€ d'achat
                        </Text>
                      </Section>
                    </td>
                    <td width="50%" style={{ paddingLeft: '8px', paddingBottom: '16px' }}>
                      <Section className="bg-gradient-to-br from-blue-50 to-cyan-50 p-5 rounded-xl border border-blue-200">
                        <Text className="text-4xl mb-2">↩️</Text>
                        <Text className="text-gray-900 font-bold text-base m-0 mb-1">
                          Retours gratuits
                        </Text>
                        <Text className="text-gray-600 text-sm m-0">
                          Sous 30 jours
                        </Text>
                      </Section>
                    </td>
                  </tr>
                  <tr>
                    <td width="50%" style={{ paddingRight: '8px' }}>
                      <Section className="bg-gradient-to-br from-yellow-50 to-orange-50 p-5 rounded-xl border border-yellow-200">
                        <Text className="text-4xl mb-2">🎟️</Text>
                        <Text className="text-gray-900 font-bold text-base m-0 mb-1">
                          Codes promo
                        </Text>
                        <Text className="text-gray-600 text-sm m-0">
                          Offres exclusives
                        </Text>
                      </Section>
                    </td>
                    <td width="50%" style={{ paddingLeft: '8px' }}>
                      <Section className="bg-gradient-to-br from-purple-50 to-pink-50 p-5 rounded-xl border border-purple-200">
                        <Text className="text-4xl mb-2">💬</Text>
                        <Text className="text-gray-900 font-bold text-base m-0 mb-1">
                          Support 7j/7
                        </Text>
                        <Text className="text-gray-600 text-sm m-0">
                          Toujours disponible
                        </Text>
                      </Section>
                    </td>
                  </tr>
                </table>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Promo Code Highlight */}
              <Section className="bg-gradient-to-r from-yellow-400 via-orange-400 to-red-400 p-8 rounded-2xl text-center my-8 shadow-xl">
                <Text className="text-white text-sm font-bold uppercase tracking-widest m-0 mb-3">
                  🎁 Offre de bienvenue
                </Text>
                <Text className="text-white text-2xl font-bold m-0 mb-4">
                  -10% sur votre première commande
                </Text>
                <Section className="bg-white p-4 rounded-xl inline-block">
                  <Text className="text-orange-600 text-3xl font-black tracking-wider m-0">
                    WELCOME10
                  </Text>
                </Section>
                <Text className="text-white text-sm m-0 mt-3 opacity-90">
                  Valable jusqu'au 31/12/2024
                </Text>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Quick Links */}
              <Section className="text-center mb-6">
                <Text className="text-gray-700 text-base font-semibold mb-4">
                  Explorez nos catégories populaires
                </Text>
                <Text className="m-0">
                  <Link
                    href="http://cedra.eldocam.com:5173/category/electronics"
                    className="text-purple-600 font-medium text-sm no-underline mx-2"
                  >
                    📱 Électronique
                  </Link>
                  {' • '}
                  <Link
                    href="http://cedra.eldocam.com:5173/category/fashion"
                    className="text-purple-600 font-medium text-sm no-underline mx-2"
                  >
                    👕 Mode
                  </Link>
                  {' • '}
                  <Link
                    href="http://cedra.eldocam.com:5173/category/home"
                    className="text-purple-600 font-medium text-sm no-underline mx-2"
                  >
                    🏠 Maison
                  </Link>
                </Text>
              </Section>
            </Section>

            {/* Footer */}
            <Section className="bg-gray-900 px-8 py-10 text-center">
              <Text className="text-white text-base font-semibold mb-4">
                Suivez-nous sur les réseaux sociaux
              </Text>
              <Text className="mb-6">
                <Link href="#" className="text-white text-2xl mx-3 no-underline">
                  📘
                </Link>
                <Link href="#" className="text-white text-2xl mx-3 no-underline">
                  📷
                </Link>
                <Link href="#" className="text-white text-2xl mx-3 no-underline">
                  �
                </Link>
                <Link href="#" className="text-white text-2xl mx-3 no-underline">
                  💼
                </Link>
              </Text>
              
              <Hr className="border-gray-700 my-6" />
              
              <Text className="text-gray-400 text-xs m-0 mb-2">
                © 2024 Cedra - Tous droits réservés
              </Text>
              <Text className="text-gray-500 text-xs m-0 mb-4">
                123 Rue du Commerce, 75001 Paris, France
              </Text>
              <Text className="text-gray-500 text-xs m-0">
                <Link href="http://cedra.eldocam.com:5173" className="text-purple-400 no-underline">
                  Visiter notre site
                </Link>
                {' • '}
                <Link href="http://cedra.eldocam.com:5173/support" className="text-purple-400 no-underline">
                  Support
                </Link>
                {' • '}
                <Link href="http://cedra.eldocam.com:5173/unsubscribe" className="text-gray-500 no-underline">
                  Se désabonner
                </Link>
              </Text>
            </Section>
          </Container>
        </Body>
      </Html>
    </Tailwind>
  );
}
