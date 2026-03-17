import React, { useEffect } from 'react';
import { 
  Box, 
  Container, 
  Typography, 
  Paper,
  Divider,
  Link,
  Accordion,
  AccordionSummary,
  AccordionDetails
} from '@mui/material';
import EmailIcon from '@mui/icons-material/Email';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { setDocumentTitle } from '../utils/titleUtils';

function About() {
  useEffect(() => {
    setDocumentTitle('关于我们');
  }, []);

  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <Paper 
        elevation={0}
        sx={{ 
          p: 4, 
          bgcolor: '#2c3e50',
          color: 'white',
          borderRadius: 2
        }}
      >
        {/* 标题部分 */}
        <Typography variant="h4" component="h1" sx={{ mb: 3, fontWeight: 'bold' }}>
          关于评课社区@BUAA
        </Typography>

        {/* 写在最前面的话 */}
        <Box sx={{ mb: 4 }}>
          <Typography variant="h5" sx={{ mb: 2, color: '#3498db', fontWeight: 'bold' }}>
            写在最前面的话
          </Typography>
          <Typography variant="body1" sx={{ mb: 2, lineHeight: 1.8 }}>
            于2025年3月在3号科研楼创建。
          </Typography>
          <Typography variant="body1" sx={{ mb: 2, lineHeight: 1.8 }}>
            第一行代码于2025年3月12日下午敲下，这是故事开始的地方，初版的最后一行代码于2025年3月18日凌晨敲完，共计用时21h56min（数据来源：WakaTime）。
          </Typography>
          <Typography variant="body1" sx={{ mb: 2, lineHeight: 1.8 }}>
            本项目全名为北京航空航天大学非官方课程测评社区，简称评课社区@BUAA（
            <Link 
              href="https://course.stuhelper.com" 
              target="_blank" 
              rel="noopener noreferrer"
              sx={{ color: '#3498db' }}
            >
              https://course.stuhelper.com
            </Link>
            ）。
          </Typography>
          <Typography variant="body1" sx={{ mb: 2, lineHeight: 1.8 }}>
            本项目的灵感来源于各高校学生自建的评课社区。
          </Typography>
          <Typography variant="body1" sx={{ mb: 2, lineHeight: 1.8 }}>
            感谢所有为本站贡献测评内容和提出意见和建议的热心同学，也感谢您的访问！
          </Typography>
        </Box>

        <Divider sx={{ my: 4, borderColor: 'rgba(255, 255, 255, 0.12)' }} />

        {/* 常见问答 (FAQ) */}
        <Box sx={{ mb: 4 }}>
          <Typography variant="h5" sx={{ mb: 3, color: '#3498db', fontWeight: 'bold' }}>
            常见问答 (FAQ)
          </Typography>
          
          <Accordion 
            sx={{ 
              mb: 2, 
              bgcolor: 'rgba(52, 73, 94, 0.7)', 
              color: 'white',
              '&:before': {
                display: 'none',
              }
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon sx={{ color: 'white' }} />}
              sx={{ borderBottom: '1px solid rgba(255, 255, 255, 0.12)' }}
            >
              <Typography variant="subtitle1" fontWeight="bold">1. 我该测评哪些课程？写一些什么？</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8 }}>
                所有的课程，尤其是尚无测评的课程。请不要吝啬你的好评，也不要害怕说坏话。即使是没有什么亮点的课程也值得你来写一条测评，因为"这门课很正常"也是很重要的信息。寥寥几字可以拯救无数人的迷茫。平台鼓励大家在写测评的时候发挥个性，自由表达。
              </Typography>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8 }}>
                当然一个理想的测评应该：
              </Typography>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8, pl: 2 }}>
                · 饱含事实。"A课很无聊"不是个好的测评，"A课的老师是PPT战士"还可以，"A课的老师会把PPT上的公式都读一遍，但是毫无解释"就非常赞了。
              </Typography>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8, pl: 2 }}>
                · 全面。"笔者上过一门课，老师讲台上声音小，讲得很紧张，但课下去问讲的paper，老师带我到办公室，用小黑板把paper全讲了一遍。虽然这门课在课堂上不一定好，但在这门课上遇到这位老师，我的收获是巨大的。"课堂、考试、作业、课后等维度都有涉及的测评是非常棒的。此外适当的幽默是最好的，不过请尊重事实。
              </Typography>
            </AccordionDetails>
          </Accordion>
          
          <Accordion 
            sx={{ 
              mb: 2, 
              bgcolor: 'rgba(52, 73, 94, 0.7)', 
              color: 'white',
              '&:before': {
                display: 'none',
              }
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon sx={{ color: 'white' }} />}
              sx={{ borderBottom: '1px solid rgba(255, 255, 255, 0.12)' }}
            >
              <Typography variant="subtitle1" fontWeight="bold">2. 本站是用来找"水课"的吗？是否会伤害教学质量？</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
                这不是本站的目的。我们希望的是提供全方位的信息，方便同学们了解某一课程的状况，包括但不限于教学内容、课堂听感、工作量、考核方式，而非哪些课容易得99+，哪些课不考勤。我们相信通过广大同学的课程测评，可以对课程和教学起到一定监督作用，从而提升教学质量。
              </Typography>
            </AccordionDetails>
          </Accordion>
          
          <Accordion 
            sx={{ 
              mb: 2, 
              bgcolor: 'rgba(52, 73, 94, 0.7)', 
              color: 'white',
              '&:before': {
                display: 'none',
              }
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon sx={{ color: 'white' }} />}
              sx={{ borderBottom: '1px solid rgba(255, 255, 255, 0.12)' }}
            >
              <Typography variant="subtitle1" fontWeight="bold">3. 为什么本站不收集和展示课程评分？是否不够直观？</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8 }}>
                我们希望弱化类似于1-10分的量化评分，主张重视文字内容，因为数据本身是个伪命题。每个同学上课的体验不一样，对于一门课是否推荐、工作量是否大的判断标准不一样，量化的数据实际上是完全主观的，而文字可以透露实质的信息。
              </Typography>
              <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
                比如某门课给分极好，均分90+，很多人满绩，那么按1-10给分，这门课可能会登顶推荐榜，但是如果没有"这门课一周两次手写C语言代码作业，老师上课一个一个点名回答问题"这样的文字测评内容，可能会坑很多人。更何况你所见到的完全是别人想让你见到的，没有人能确保数据的真实性、准确性，数据会骗人，内容才是核心。
              </Typography>
            </AccordionDetails>
          </Accordion>
          
          <Accordion 
            sx={{ 
              mb: 2, 
              bgcolor: 'rgba(52, 73, 94, 0.7)', 
              color: 'white',
              '&:before': {
                display: 'none',
              }
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon sx={{ color: 'white' }} />}
              sx={{ borderBottom: '1px solid rgba(255, 255, 255, 0.12)' }}
            >
              <Typography variant="subtitle1" fontWeight="bold">4. 本站由谁创建？由谁管理？</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
                不知道，但大概率是在校生吧。我们希望始终保持这个小站的绝对匿名和绝对真实透明，为了防止受到外界因素干扰，请不要在任何场合和平台开盒，谢谢！否则我们可能会不得不关停本站。
              </Typography>
            </AccordionDetails>
          </Accordion>
          
          <Accordion 
            sx={{ 
              mb: 2, 
              bgcolor: 'rgba(52, 73, 94, 0.7)', 
              color: 'white',
              '&:before': {
                display: 'none',
              }
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon sx={{ color: 'white' }} />}
              sx={{ borderBottom: '1px solid rgba(255, 255, 255, 0.12)' }}
            >
              <Typography variant="subtitle1" fontWeight="bold">5. 我会因为在这里发表信息而被约谈吗？</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8 }}>
                参见【必要声明】：本站不设登录验证，不记录访问者信息，这意味着本站是完全匿名的。我们希望匿名制能带给大家足够的安全感，自由表达真实而客观的观点。
              </Typography>
              <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
                首先我们并不希望有人能找到我们，否则我们可能会受到外界因素干扰而不得不做出一些不利于本站良性发展直至关停本站的措施。即使有人找到我们，我们也能保证包括开发者在内的任何单位或个人都无法获取访问者和内容发布者的任何信息（除非你在课程测评中自报家门）。
              </Typography>
            </AccordionDetails>
          </Accordion>
          
          <Accordion 
            sx={{ 
              mb: 2, 
              bgcolor: 'rgba(52, 73, 94, 0.7)', 
              color: 'white',
              '&:before': {
                display: 'none',
              }
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon sx={{ color: 'white' }} />}
              sx={{ borderBottom: '1px solid rgba(255, 255, 255, 0.12)' }}
            >
              <Typography variant="subtitle1" fontWeight="bold">6. 网站上的内容是否受管理？</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8 }}>
                参见【必要声明】：我们坚持信息原生态、公开、透明的原则，不对课程测评进行审查、修改、屏蔽，当然也不对内容的真实性负责。
              </Typography>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8 }}>
                当然这不意味着我们对内容完全不加管理，以下内容本站会进行删改：
              </Typography>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8, pl: 2 }}>
                ①非课程测评信息。比如无意义的垃圾灌水、全篇攻击某学院或教师等等，尺度由本站管理者自行把握，也请内容发布者自行酌定。
              </Typography>
              <Typography variant="body1" paragraph sx={{ lineHeight: 1.8, pl: 2 }}>
                ②侵犯其他单位或个人权益的内容。比如暴露无意义不必要的隐私、与课程内容无关的诽谤（例："A老师没来上课，因为他的车胎爆了，这是今年第二次爆胎了""A老师是个醉汉，天天上课跟喝高了一样"都不被接受，但是"A老师偶尔迟到，上课迷迷糊糊，有时自己也不知道自己说到哪了"可以）。
              </Typography>
              <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
                与事实不符，经本人/他人举报并举证查实的内容会被删改，但是我们尽量不采取这样的措施。如果您认为某一课程测评背离事实或有失偏颇，您可以在测评的右下角点击[反对]，也就是倒拇指图标，并发布测评表达您自己的观点，但请不要攻击他人。
              </Typography>
            </AccordionDetails>
          </Accordion>
          
          <Accordion 
            sx={{ 
              mb: 2, 
              bgcolor: 'rgba(52, 73, 94, 0.7)', 
              color: 'white',
              '&:before': {
                display: 'none',
              }
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon sx={{ color: 'white' }} />}
              sx={{ borderBottom: '1px solid rgba(255, 255, 255, 0.12)' }}
            >
              <Typography variant="subtitle1" fontWeight="bold">7. 网站上的内容版权问题如何界定？</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
                向本站提交测评时，您同意向本站提供对您提交的测评内容的永久的（perpetual）、不可撤回的（irrevocable）、非独占的（non-exclusive）、全球有效的（worldwide）无限制的许可（license）。您依然享有您提交的内容的全部版权。任何人（除了原始所有权人）不得在未经许可的条件下，转载本站内容（包括全部或部分内容）。您在本站提交测评时，授权本站以一切法律、技术途径代理保护您所提交的内容的版权，包括但不限于向任何未经许可的转载媒介进行起诉和发送DMCA请求。
              </Typography>
            </AccordionDetails>
          </Accordion>
        </Box>

        <Divider sx={{ my: 4, borderColor: 'rgba(255, 255, 255, 0.12)' }} />

        {/* 联系我们 */}
        <Box sx={{ mb: 4 }}>
          <Typography variant="h5" sx={{ mb: 2, color: '#3498db', fontWeight: 'bold' }}>
            联系我们
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
            <EmailIcon sx={{ mr: 1, color: '#3498db' }} />
            <Link 
              href="mailto:stuhelper@protonmail.com"
              sx={{ color: '#3498db', textDecoration: 'none' }}
            >
              stuhelper@protonmail.com
            </Link>
          </Box>
          <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
            欢迎各位同学与老师向本站提供宝贵的意见和建议。如果您是老师，也欢迎您对课程测评的疑问或质疑进行答复或澄清。
          </Typography>
        </Box>

        <Divider sx={{ my: 4, borderColor: 'rgba(255, 255, 255, 0.12)' }} />

        {/* 必要声明 */}
        <Box>
          <Typography variant="h5" sx={{ mb: 2, color: '#3498db', fontWeight: 'bold' }}>
            必要声明
          </Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
              本评课社区为非官方网站，与北京航空航天大学及其下设机关或组织没有任何关联。
            </Typography>
            <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
              我们坚持信息原生态、公开、透明的原则，不对课程测评进行审查、修改、屏蔽，当然也不对内容的真实性负责——你所见到的完全是别人敲击键盘打下的。如果您认为某一课程测评背离事实或有失偏颇，您可以在测评的右下角点击"反对"，也就是倒拇指图标，并发布测评表达您自己的观点，但请不要攻击他人。我们相信全面的信息会给大家最好的答案。
            </Typography>
            <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
              本站不设登录验证，不记录访问者信息，这意味着本站是完全匿名的。我们希望匿名制能带给大家足够的安全感，自由表达真实而客观的观点。
            </Typography>
            <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
              本站运营者的责任仅限于维护系统的稳定、删除垃圾灌水信息、更新课程列表。
            </Typography>
            <Typography variant="body1" sx={{ lineHeight: 1.8 }}>
              即便如此，我们希望所有访问者能共同维护这个小站，保证内容的丰富度和合理性，保持本站创建的初衷——"帮你找到最适合的课程"。
            </Typography>
          </Box>
        </Box>
      </Paper>
    </Container>
  );
}

export default About; 