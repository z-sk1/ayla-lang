import clsx from "clsx";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

export const FeatureList = [
  {
    title: "Simple Syntax",
    image: require("@site/static/img/code.png").default,
    description: (
      <>
        Ayla focuses on readability and simplicity. The syntax is predictable,
        and easy to understand.
      </>
    ),
  },
  {
    title: "Static Typing",
    image: require("@site/static/img/shield.png").default,
    description: (
      <>
        Ayla uses static typing to catch errors early while keeping the language
        lightweight.
      </>
    ),
  },
  {
    title: "Powered by Go",
    image: require("@site/static/img/go.png").default,
    description: (
      <>Ayla is built with Go and mirrors many of its philosophies.</>
    ),
  },
];

function Feature({ image, title, description }) {
  return (
    <div className={clsx("col col--4")}>
      <div className="text--center">
        <img src={image} className={styles.featureSvg} alt={title} />
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
