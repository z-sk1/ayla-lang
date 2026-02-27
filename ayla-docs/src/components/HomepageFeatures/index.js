import clsx from "clsx";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

export const FeatureList = [
  {
    title: "Simple Syntax",
    Svg: require("@site/static/img/logo.svg").default,
    description: (
      <>
        Ayla focuses on readability and simplicity. The syntax is inspired by
        Go, keeping things explicit, predictable, and easy to understand.
      </>
    ),
  },
  {
    title: "Static Typing",
    Svg: require("@site/static/img/logo.svg").default,
    description: (
      <>
        Ayla uses static typing to catch errors early while keeping the language
        lightweight.
      </>
    ),
  },
  {
    title: "Powered by Go",
    Svg: require("@site/static/img/go.svg").default,
    description: (
      <>
        Ayla is built with Go and it mirrors alot of the philosophies and
        patterns found in it.
      </>
    ),
  },
];

function Feature({ Svg, title, description }) {
  return (
    <div className={clsx("col col--4")}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
