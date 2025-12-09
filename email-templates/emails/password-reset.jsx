import React from 'react';
import {
  Html,
  Head,
  Preview,
  Body,
  Container,
  Section,
  Heading,
  Text,
  Button,
  Link,
  Hr,
  Tailwind,
} from '@react-email/components';

export default function PasswordResetEmail({
  userName = 'Utilisateur',
  resetToken = 'abc123xyz789',
  expiresIn = '1 heure',
}) {
  const resetUrl = `http://cedra.eldocam.com:5173/reset-password?token=${resetToken}`;

  return (
    <Tailwind>
      <Html lang="fr">
        <Head />
        <Preview>🔐 Réinitialisation de votre mot de passe Cedra</Preview>
        <Body className="bg-gray-50 font-sans">
          <Container className="max-w-2xl mx-auto my-10 bg-white rounded-2xl shadow-2xl overflow-hidden">
            
            {/* Header */}
            <Section className="bg-gradient-to-r from-orange-500 to-red-500 p-10 text-center">
              <Text className="text-6xl mb-4">🔐</Text>
              <Heading className="text-white text-3xl font-bold m-0 mb-2">
                Réinitialisation du mot de passe
              </Heading>
              <Text className="text-white text-lg m-0 opacity-90">
                Sécurisez votre compte Cedra
              </Text>
            </Section>

            {/* Content */}
            <Section className="px-8 py-10">
              <Text className="text-gray-800 text-lg leading-relaxed mb-2">
                Bonjour <strong>{userName}</strong>,
              </Text>

              <Text className="text-gray-700 text-base leading-relaxed mb-6">
                Nous avons reçu une demande de réinitialisation de mot de passe pour votre compte Cedra. Si vous n'êtes pas à l'origine de cette demande, vous pouvez ignorer cet email en toute sécurité.
              </Text>

              {/* Warning Box */}
              <Section className="bg-red-50 border-l-4 border-red-500 p-5 rounded-lg mb-8">
                <Text className="text-red-900 font-semibold text-base m-0 mb-2">
                  ⚠️ Important
                </Text>
                <Text className="text-red-800 text-sm m-0">
                  Ce lien est valable pendant <strong>{expiresIn}</strong> uniquement. Ne partagez jamais ce lien avec personne.
                </Text>
              </Section>

              {/* CTA Button */}
              <Section className="text-center my-10">
                <Button
                  href={resetUrl}
                  className="bg-gradient-to-r from-orange-500 to-red-500 text-white px-10 py-4 rounded-xl font-bold text-lg no-underline inline-block shadow-lg"
                >
                  🔑 Réinitialiser mon mot de passe
                </Button>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Alternative Link */}
              <Section className="bg-gray-50 p-6 rounded-xl mb-8">
                <Text className="text-gray-700 text-sm m-0 mb-3">
                  Si le bouton ne fonctionne pas, copiez et collez ce lien dans votre navigateur :
                </Text>
                <Text className="bg-white px-4 py-3 rounded text-orange-600 text-xs m-0" style={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>
                  {resetUrl}
                </Text>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Security Tips */}
              <Section className="mb-8">
                <Heading className="text-gray-900 text-xl font-bold mb-4">
                  🛡️ Conseils de sécurité
                </Heading>

                <table width="100%" cellPadding="0" cellSpacing="0">
                  <tr>
                    <td width="30" style={{ verticalAlign: 'top', paddingTop: '4px' }}>
                      <Text className="text-green-500 text-2xl m-0">✓</Text>
                    </td>
                    <td style={{ paddingBottom: '12px' }}>
                      <Text className="text-gray-700 text-sm m-0">
                        Utilisez un mot de passe unique et complexe (12+ caractères)
                      </Text>
                    </td>
                  </tr>
                  <tr>
                    <td width="30" style={{ verticalAlign: 'top', paddingTop: '4px' }}>
                      <Text className="text-green-500 text-2xl m-0">✓</Text>
                    </td>
                    <td style={{ paddingBottom: '12px' }}>
                      <Text className="text-gray-700 text-sm m-0">
                        Mélangez majuscules, minuscules, chiffres et symboles
                      </Text>
                    </td>
                  </tr>
                  <tr>
                    <td width="30" style={{ verticalAlign: 'top', paddingTop: '4px' }}>
                      <Text className="text-green-500 text-2xl m-0">✓</Text>
                    </td>
                    <td style={{ paddingBottom: '12px' }}>
                      <Text className="text-gray-700 text-sm m-0">
                        N'utilisez jamais le même mot de passe sur plusieurs sites
                      </Text>
                    </td>
                  </tr>
                  <tr>
                    <td width="30" style={{ verticalAlign: 'top', paddingTop: '4px' }}>
                      <Text className="text-green-500 text-2xl m-0">✓</Text>
                    </td>
                    <td>
                      <Text className="text-gray-700 text-sm m-0">
                        Activez l'authentification à deux facteurs si disponible
                      </Text>
                    </td>
                  </tr>
                </table>
              </Section>

              <Hr className="border-gray-200 my-8" />

              {/* Help Section */}
              <Section className="bg-blue-50 p-6 rounded-xl border border-blue-200">
                <Text className="text-blue-900 font-semibold text-base m-0 mb-2">
                  🤔 Vous n'avez pas demandé cette réinitialisation ?
                </Text>
                <Text className="text-blue-800 text-sm m-0 mb-4">
                  Si vous n'êtes pas à l'origine de cette demande, votre compte pourrait être compromis. Contactez immédiatement notre équipe de sécurité.
                </Text>
                <Link
                  href="http://cedra.eldocam.com:5173/support/security"
                  className="text-blue-600 font-semibold text-sm no-underline"
                >
                  Signaler un problème de sécurité →
                </Link>
              </Section>
            </Section>

            {/* Footer */}
            <Section className="bg-gray-900 px-8 py-8 text-center">
              <Text className="text-gray-400 text-xs m-0 mb-2">
                © 2024 Cedra - Tous droits réservés
              </Text>
              <Text className="text-gray-500 text-xs m-0 mb-4">
                Cet email a été envoyé automatiquement, merci de ne pas y répondre.
              </Text>
              <Text className="text-gray-500 text-xs m-0">
                <Link href="http://cedra.eldocam.com:5173" className="text-purple-400 no-underline">
                  Visiter notre site
                </Link>
                {' • '}
                <Link href="http://cedra.eldocam.com:5173/support" className="text-purple-400 no-underline">
                  Support
                </Link>
              </Text>
            </Section>
          </Container>
        </Body>
      </Html>
    </Tailwind>
  );
}
